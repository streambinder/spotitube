package provider

import (
	"crypto/md5" //nolint:gosec // md5 is the signature scheme mandated by the qobuz API, not a security control
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

const qobuzSearchResponse = `{"tracks":{"items":[{"id":138731318,"performer":{"name":"Artist"}}]}}`

const (
	qobuzShellHTML     = `<html><head><script src="/resources/1.0/bundle.js"></script></head></html>`
	qobuzBundleJS      = `var config={production:{api:{appId:"123456789",appSecret:"00000000000000000000000000000000"}}};`
	qobuzOpenShellHTML = `<html><head><script src="/resources/1.0/js/main.js"></script></head></html>`
	qobuzOpenBundleJS  = `var config={app_id:"123456789",app_secret:"00000000000000000000000000000000"};`
	qobuzFileJSON      = `{"url":"https://cdn.qobuz.example/track.mp3","sample":false}`
)

// setQobuzToken installs a dummy user token, so the file URL stage can
// run; the token value itself is never asserted.
func setQobuzToken(t *testing.T) {
	t.Helper()
	t.Setenv(qobuzTokenEnv, "test-token")
}

// qobuzScript scripts the four HTTP stages of the qobuz flow: the shell
// page, the JS bundle carrying the credentials, the catalog search and
// the getFileUrl resolution. Zero fields select the successful default
// response.
type qobuzScript struct {
	shellStatus   int
	shellBody     string
	shellErr      error
	shellReadErr  bool
	bundleStatus  int
	bundleBody    string
	bundleErr     error
	bundleReadErr bool
	searchStatus  int
	searchBody    string
	searchErr     error
	fileStatus    int
	fileBody      string
	fileErr       error
}

func qobuzStage(status int, body string, err error, readErr bool) (*http.Response, error) {
	if err != nil {
		return nil, err
	}
	if status == 0 {
		status = 200
	}
	response := httpResponse(status, body)
	if readErr {
		response.Body = errReader{errors.New("ko")}
	}
	return response, nil
}

// qobuzCredStage answers a shell or bundle request of either
// credential scrape, with per-app default bodies.
func qobuzCredStage(script qobuzScript, shell bool, shellDefault, bundleDefault string) (*http.Response, error) {
	if shell {
		body := script.shellBody
		if body == "" {
			body = shellDefault
		}
		return qobuzStage(script.shellStatus, body, script.shellErr, script.shellReadErr)
	}
	body := script.bundleBody
	if body == "" {
		body = bundleDefault
	}
	return qobuzStage(script.bundleStatus, body, script.bundleErr, script.bundleReadErr)
}

// qobuzFlow answers a request of any qobuz stage according to the script.
func qobuzFlow(request *http.Request, script qobuzScript) (*http.Response, error) {
	switch {
	case request.URL.Host == "open.qobuz.com" && request.URL.Path == "/track/1":
		return qobuzCredStage(script, true, qobuzOpenShellHTML, qobuzOpenBundleJS)
	case request.URL.Host == "open.qobuz.com":
		return qobuzCredStage(script, false, qobuzOpenShellHTML, qobuzOpenBundleJS)
	case request.URL.Host == "play.qobuz.com" && request.URL.Path == "/login":
		return qobuzCredStage(script, true, qobuzShellHTML, qobuzBundleJS)
	case request.URL.Host == "play.qobuz.com":
		return qobuzCredStage(script, false, qobuzShellHTML, qobuzBundleJS)
	case request.URL.Host == "www.qobuz.com" && strings.Contains(request.URL.Path, "getFileUrl"):
		body := script.fileBody
		if body == "" {
			body = qobuzFileJSON
		}
		return qobuzStage(script.fileStatus, body, script.fileErr, false)
	case request.URL.Host == "www.qobuz.com":
		body := script.searchBody
		if body == "" {
			body = qobuzSearchResponse
		}
		return qobuzStage(script.searchStatus, body, script.searchErr, false)
	}
	return httpResponse(404, ""), nil
}

// resetQobuz clears the credentials and stream URL caches, so each
// test starts from a cold qobuz state regardless of test order.
func resetQobuz(t *testing.T) {
	t.Helper()
	reset := func() {
		qobuzCredMu.Lock()
		defer qobuzCredMu.Unlock()
		qobuzCachedID = ""
		qobuzCachedSecrets = nil
		qobuzFileSecret = ""
		qobuzSearchCachedID = ""
		qobuzSearchCachedSecret = ""
		qobuzURLCache = sync.Map{}
	}
	reset()
	t.Cleanup(reset)
}

func BenchmarkQobuz(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestQobuzSearch(&testing.T{})
	}
}

func TestQobuzSearch(t *testing.T) {
	resetQobuz(t)
	setQobuzToken(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{})
	})

	matches, err := qobuz{}.search(track)
	assert.Nil(t, err)
	assert.Len(t, matches, 1)
	assert.Equal(t, 100, matches[0].Score)
	assert.Equal(t, "https://cdn.qobuz.example/track.mp3", matches[0].URL)
}

func TestQobuzSearchCacheHit(t *testing.T) {
	resetQobuz(t)
	qobuzURLCache.Store("138731318", "https://cached.example/track.mp3")
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		// the file stage must never be reached: the URL cache answers first
		return qobuzFlow(request, qobuzScript{fileErr: errors.New("must not be called")})
	})

	matches, err := qobuz{}.search(track)
	assert.Nil(t, err)
	assert.Len(t, matches, 1)
	assert.Equal(t, "https://cached.example/track.mp3", matches[0].URL)
}

func TestQobuzSearchCredentialsFailure(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{shellErr: errors.New("ko")})
	})

	matches, err := qobuz{}.search(track)
	assert.NotNil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchRequestFailure(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{searchErr: errors.New("ko")})
	})

	matches, err := qobuz{}.search(track)
	assert.NotNil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchNonOKStatus(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{searchStatus: 500})
	})

	matches, err := qobuz{}.search(track)
	assert.NotNil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchMalformedResponse(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{searchBody: `{not json}`})
	})

	matches, err := qobuz{}.search(track)
	assert.NotNil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchNoItems(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{searchBody: `{"tracks":{"items":[]}}`})
	})

	matches, err := qobuz{}.search(track)
	assert.Nil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchNoToken(t *testing.T) {
	resetQobuz(t)
	t.Setenv(qobuzTokenEnv, "")
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{})
	})

	matches, err := qobuz{}.search(track)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), qobuzTokenEnv)
	assert.Nil(t, matches)
}

func TestQobuzSearchFileFailure(t *testing.T) {
	resetQobuz(t)
	setQobuzToken(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{fileErr: errors.New("ko")})
	})

	matches, err := qobuz{}.search(track)
	assert.NotNil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchFileBadJSON(t *testing.T) {
	resetQobuz(t)
	setQobuzToken(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{fileBody: "not json"})
	})

	matches, err := qobuz{}.search(track)
	assert.NotNil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchFileEmptyURL(t *testing.T) {
	resetQobuz(t)
	setQobuzToken(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{fileBody: `{"url":"","sample":false}`})
	})

	matches, err := qobuz{}.search(track)
	assert.NotNil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchFileSample(t *testing.T) {
	resetQobuz(t)
	setQobuzToken(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{fileBody: `{"url":"https://cdn.qobuz.example/sample.mp3","sample":true,"restrictions":[{"code":"UserUnauthenticated"}]}`})
	})

	matches, err := qobuz{}.search(track)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "sample")
	assert.Contains(t, err.Error(), "UserUnauthenticated")
	assert.Nil(t, matches)
}

func TestQobuzSearchFileNonOKStatus(t *testing.T) {
	resetQobuz(t)
	setQobuzToken(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{fileStatus: 500, fileBody: "error page"})
	})

	matches, err := qobuz{}.search(track)
	assert.NotNil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzCredentials(t *testing.T) {
	resetQobuz(t)
	callCount := 0
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		callCount++
		return qobuzFlow(request, qobuzScript{})
	})

	id, secret, err := qobuzFileCredentials()
	assert.Nil(t, err)
	assert.Equal(t, "123456789", id)
	assert.Equal(t, []string{"00000000000000000000000000000000"}, secret)

	// the second call is served by the credentials cache: no new requests
	id2, secret2, err2 := qobuzFileCredentials()
	assert.Nil(t, err2)
	assert.Equal(t, id, id2)
	assert.Equal(t, secret, secret2)
	assert.Equal(t, 2, callCount)
}

func TestQobuzCredentialsShellFailure(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{shellErr: errors.New("ko")})
	})

	_, _, err := qobuzFileCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsShellNonOK(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{shellStatus: 500})
	})

	_, _, err := qobuzFileCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsNoBundleScript(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{shellBody: "<html><head></head></html>"})
	})

	_, _, err := qobuzFileCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsBundleFailure(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{bundleErr: errors.New("ko")})
	})

	_, _, err := qobuzFileCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsBundleNonOK(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{bundleStatus: 500})
	})

	_, _, err := qobuzFileCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsNoCredentialsInBundle(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{bundleBody: "var config={};"})
	})

	_, _, err := qobuzFileCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsShellReadFailure(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{shellReadErr: true})
	})

	_, _, err := qobuzFileCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsBundleRequestBuildFailure(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		// a bundle URL with a control character makes http.NewRequest fail
		return qobuzFlow(request, qobuzScript{shellBody: "<script src=\"/resources/\x7f/bundle.js\"></script>"})
	})

	_, _, err := qobuzFileCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsBundleReadFailure(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{bundleReadErr: true})
	})

	_, _, err := qobuzFileCredentials()
	assert.NotNil(t, err)
}

func TestQobuzFileURL(t *testing.T) {
	resetQobuz(t)
	setQobuzToken(t)
	callCount := 0
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		callCount++
		return qobuzFlow(request, qobuzScript{fileBody: `{"url":"https://cdn.example/track.mp3","sample":false}`})
	})

	fileURL, err := qobuzFileURL("424242")
	assert.Nil(t, err)
	assert.Equal(t, "https://cdn.example/track.mp3", fileURL)

	// the second resolution is served by the URL cache: the first one
	// took three calls (shell, bundle, getFileUrl), the second none
	fileURL2, err2 := qobuzFileURL("424242")
	assert.Nil(t, err2)
	assert.Equal(t, fileURL, fileURL2)
	assert.Equal(t, 3, callCount)
}

func TestQobuzFileURLFailure(t *testing.T) {
	resetQobuz(t)
	setQobuzToken(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{fileErr: errors.New("ko")})
	})

	_, err := qobuzFileURL("434343")
	assert.NotNil(t, err)
}

// qobuzSeededBundle renders a web player bundle whose seed data
// derives two candidate secrets: berlin, the second timezone found
// and therefore the first candidate, decodes to "berlin-secret";
// london decodes to "test-secret-value".
func qobuzSeededBundle() string {
	return `var config={production:{api:{appId:"123456789",appSecret:"00000000000000000000000000000000"}}};` +
		`a.initialSeed("dGVzdC1z",window.utimezone.london);` +
		`b.initialSeed("YmVybGlu",window.utimezone.berlin);` +
		`name:"app/London",info:"ZWNyZXQt",extras:"dmFsdWU=` + strings.Repeat("A", 44) + `";` +
		`name:"app/Berlin",info:"LXNlY3Jl",extras:"dA==` + strings.Repeat("B", 44) + `";`
}

func TestQobuzDeriveSecrets(t *testing.T) {
	assert.Equal(t,
		[]string{"berlin-secret", "test-secret-value"},
		qobuzDeriveSecrets([]byte(qobuzSeededBundle())))
	assert.Empty(t, qobuzDeriveSecrets([]byte(qobuzBundleJS)))
}

func TestQobuzFileURLSecondCandidate(t *testing.T) {
	resetQobuz(t)
	setQobuzToken(t)
	attempts := 0
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		if request.URL.Host == "www.qobuz.com" {
			attempts++
			query := request.URL.Query()
			sigFor := func(secret string) string {
				sigBytes := md5.Sum([]byte("trackgetFileUrl" + "format_id" + "5" +
					"intent" + "stream" + "track_id" + "424242" + query.Get("request_ts") + secret))
				return fmt.Sprintf("%x", sigBytes)
			}
			switch query.Get("request_sig") {
			case sigFor("berlin-secret"):
				// the API refuses a signature built with the wrong candidate
				return httpResponse(400, `{}`), nil
			case sigFor("test-secret-value"):
				return httpResponse(200, `{"url":"https://cdn.example/track.mp3","sample":false}`), nil
			}
			return httpResponse(500, `{}`), nil
		}
		return qobuzFlow(request, qobuzScript{bundleBody: qobuzSeededBundle()})
	})

	// testing: the first candidate is rejected, the second resolves,
	// and the accepted secret is remembered
	fileURL, err := qobuzFileURL("424242")
	assert.Nil(t, err)
	assert.Equal(t, "https://cdn.example/track.mp3", fileURL)
	assert.Equal(t, 2, attempts)
	assert.Equal(t, "test-secret-value", qobuzFileSecret)
}
