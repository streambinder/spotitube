package provider

import (
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

const qobuzSearchResponse = `{"tracks":{"items":[{"id":138731318,"performer":{"name":"Artist"}}]}}`

const (
	qobuzShellHTML = `<html><head><script src="/resources/1.0/js/main.js"></script></head></html>`
	qobuzBundleJS  = `var config={app_id:"123456789",app_secret:"00000000000000000000000000000000"};`
	qobuzProxyJSON = `{"url":"https://cdn.qobuz.example/track.mp3"}`
)

// qobuzScript scripts the four HTTP stages of the qobuz flow: the shell
// page, the JS bundle carrying the credentials, the catalog search and
// the CDN proxy. Zero fields select the successful default response.
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
	proxyStatus   int
	proxyBody     string
	proxyErr      error
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

// qobuzFlow answers a request of any qobuz stage according to the script.
func qobuzFlow(request *http.Request, script qobuzScript) (*http.Response, error) {
	switch {
	case request.URL.Host == "open.qobuz.com" && request.URL.Path == "/track/1":
		body := script.shellBody
		if body == "" {
			body = qobuzShellHTML
		}
		return qobuzStage(script.shellStatus, body, script.shellErr, script.shellReadErr)
	case request.URL.Host == "open.qobuz.com":
		body := script.bundleBody
		if body == "" {
			body = qobuzBundleJS
		}
		return qobuzStage(script.bundleStatus, body, script.bundleErr, script.bundleReadErr)
	case request.URL.Host == "www.qobuz.com":
		body := script.searchBody
		if body == "" {
			body = qobuzSearchResponse
		}
		return qobuzStage(script.searchStatus, body, script.searchErr, false)
	case request.URL.Host == "dabmusic.xyz":
		body := script.proxyBody
		if body == "" {
			body = qobuzProxyJSON
		}
		return qobuzStage(script.proxyStatus, body, script.proxyErr, false)
	}
	return httpResponse(404, ""), nil
}

// resetQobuz clears the credentials and CDN caches, so each test starts
// from a cold qobuz state regardless of test order.
func resetQobuz(t *testing.T) {
	t.Helper()
	reset := func() {
		qobuzCredMu.Lock()
		defer qobuzCredMu.Unlock()
		qobuzCachedID = ""
		qobuzCachedSecret = ""
		qobuzCDNCache = sync.Map{}
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
	qobuzCDNCache.Store("138731318", "https://cached.example/track.mp3")
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		// the proxy stage must never be reached: the CDN cache answers first
		return qobuzFlow(request, qobuzScript{proxyErr: errors.New("must not be called")})
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

func TestQobuzSearchAllProxiesFailed(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{proxyErr: errors.New("proxy down")})
	})

	matches, err := qobuz{}.search(track)
	assert.NotNil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchProxyBadJSON(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{proxyBody: "not json"})
	})

	matches, err := qobuz{}.search(track)
	assert.NotNil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchProxyEmptyURL(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{proxyBody: `{"url":""}`})
	})

	matches, err := qobuz{}.search(track)
	assert.NotNil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchProxyNonOKStatus(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{proxyStatus: 500, proxyBody: "error page"})
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

	id, secret, err := qobuzCredentials()
	assert.Nil(t, err)
	assert.Equal(t, "123456789", id)
	assert.Equal(t, "00000000000000000000000000000000", secret)

	// the second call is served by the credentials cache: no new requests
	id2, secret2, err2 := qobuzCredentials()
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

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsShellNonOK(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{shellStatus: 500})
	})

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsNoBundleScript(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{shellBody: "<html><head></head></html>"})
	})

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsBundleFailure(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{bundleErr: errors.New("ko")})
	})

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsBundleNonOK(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{bundleStatus: 500})
	})

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsNoCredentialsInBundle(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{bundleBody: "var config={};"})
	})

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsShellReadFailure(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{shellReadErr: true})
	})

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsBundleRequestBuildFailure(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		// a bundle URL with a control character makes http.NewRequest fail
		return qobuzFlow(request, qobuzScript{shellBody: "<script src=\"/resources/\x7f/js/main.js\"></script>"})
	})

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsBundleReadFailure(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{bundleReadErr: true})
	})

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCDNURL(t *testing.T) {
	resetQobuz(t)
	callCount := 0
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		callCount++
		return qobuzFlow(request, qobuzScript{proxyBody: `{"url":"https://cdn.example/track.mp3"}`})
	})

	cdnURL, err := qobuzCDNURL("424242")
	assert.Nil(t, err)
	assert.Equal(t, "https://cdn.example/track.mp3", cdnURL)

	// the second resolution is served by the CDN cache
	cdnURL2, err2 := qobuzCDNURL("424242")
	assert.Nil(t, err2)
	assert.Equal(t, cdnURL, cdnURL2)
	assert.Equal(t, 1, callCount)
}

func TestQobuzCDNURLAllFailed(t *testing.T) {
	resetQobuz(t)
	stubHTTP(t, func(request *http.Request) (*http.Response, error) {
		return qobuzFlow(request, qobuzScript{proxyErr: errors.New("proxy down")})
	})

	_, err := qobuzCDNURL("434343")
	assert.NotNil(t, err)
}
