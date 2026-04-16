package provider

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"
)

const qobuzSearchResponse = `{"tracks":{"items":[{"id":138731318,"performer":{"name":"Artist"}}]}}`

func BenchmarkQobuz(b *testing.B) {
	for b.Loop() {
		TestQobuzSearch(&testing.T{})
	}
}

// mockQobuzSearch bypasses credential fetching and mocks only the search+proxy calls
func mockQobuzSearch(searchBody, proxyBody string, searchStatus, proxyStatus int) {
	mockey.Mock(qobuzCredentials).Return("appid", "appsecret", nil).Build()
	callCount := 0
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Do")).To(func(_ *http.Request) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			return &http.Response{StatusCode: searchStatus, Body: io.NopCloser(strings.NewReader(searchBody))}, nil
		}
		return &http.Response{StatusCode: proxyStatus, Body: io.NopCloser(strings.NewReader(proxyBody))}, nil
	}).Build()
}

func TestQobuzSearch(t *testing.T) {
	defer mockey.UnPatchAll()
	mockQobuzSearch(qobuzSearchResponse, `{"url":"https://cdn.qobuz.example/track.mp3"}`, 200, 200)

	matches, err := qobuz{}.search(track)
	assert.Nil(t, err)
	assert.Len(t, matches, 1)
	assert.Equal(t, 100, matches[0].Score)
	assert.Equal(t, "https://cdn.qobuz.example/track.mp3", matches[0].URL)
}

func TestQobuzSearchCredentialsFailure(t *testing.T) {
	defer mockey.UnPatchAll()
	mockey.Mock(qobuzCredentials).Return("", "", errors.New("ko")).Build()

	matches, err := qobuz{}.search(track)
	assert.Nil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchRequestBuildFailure(t *testing.T) {
	defer mockey.UnPatchAll()
	mockey.Mock(qobuzCredentials).Return("appid", "appsecret", nil).Build()
	mockey.Mock(http.NewRequest).Return(nil, errors.New("ko")).Build()

	matches, err := qobuz{}.search(track)
	assert.Nil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchRequestFailure(t *testing.T) {
	defer mockey.UnPatchAll()
	mockey.Mock(qobuzCredentials).Return("appid", "appsecret", nil).Build()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Do")).Return(nil, errors.New("ko")).Build()

	matches, err := qobuz{}.search(track)
	assert.Nil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchNonOKStatus(t *testing.T) {
	defer mockey.UnPatchAll()
	mockQobuzSearch("", "", 500, 0)

	matches, err := qobuz{}.search(track)
	assert.Nil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchMalformedResponse(t *testing.T) {
	defer mockey.UnPatchAll()
	mockQobuzSearch(`{not json}`, "", 200, 0)

	// malformed search response: qobuzSearchTrack returns error, search swallows it (non-fatal)
	matches, err := qobuz{}.search(track)
	assert.Nil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchNoItems(t *testing.T) {
	defer mockey.UnPatchAll()
	mockQobuzSearch(`{"tracks":{"items":[]}}`, "", 200, 0)

	matches, err := qobuz{}.search(track)
	assert.Nil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchAllProxiesFailed(t *testing.T) {
	defer mockey.UnPatchAll()
	mockey.Mock(qobuzCredentials).Return("appid", "appsecret", nil).Build()
	callCount := 0
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Do")).To(func(_ *http.Request) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(qobuzSearchResponse))}, nil
		}
		return nil, errors.New("proxy down")
	}).Build()

	matches, err := qobuz{}.search(track)
	assert.Nil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchProxyBadJSON(t *testing.T) {
	defer mockey.UnPatchAll()
	mockQobuzSearch(qobuzSearchResponse, `{not json}`, 200, 200)

	matches, err := qobuz{}.search(track)
	assert.Nil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchProxyEmptyURL(t *testing.T) {
	defer mockey.UnPatchAll()
	mockQobuzSearch(qobuzSearchResponse, `{"url":""}`, 200, 200)

	matches, err := qobuz{}.search(track)
	assert.Nil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzSearchProxyNonOKStatus(t *testing.T) {
	defer mockey.UnPatchAll()
	mockQobuzSearch(qobuzSearchResponse, "", 200, 503)

	matches, err := qobuz{}.search(track)
	assert.Nil(t, err)
	assert.Nil(t, matches)
}

func TestQobuzCredentials(t *testing.T) {
	defer mockey.UnPatchAll()
	// reset cache so this test actually exercises the scraping path
	qobuzCachedID = ""
	qobuzCachedSecret = ""
	callCount := 0
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Do")).To(func(_ *http.Request) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			// shell page with bundle script tag
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`<script src="/resources/1.0/js/main.js"></script>`)),
			}, nil
		}
		// bundle with credentials
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`app_id:"123456789",app_secret:"abcdef1234567890abcdef1234567890"`)), // gitleaks:allow
		}, nil
	}).Build()

	id, secret, err := qobuzCredentials()
	assert.Nil(t, err)
	assert.Equal(t, "123456789", id)
	assert.Equal(t, "abcdef1234567890abcdef1234567890", secret)

	// second call uses cache, no extra HTTP calls
	id2, secret2, err2 := qobuzCredentials()
	assert.Nil(t, err2)
	assert.Equal(t, id, id2)
	assert.Equal(t, secret, secret2)
	assert.Equal(t, 2, callCount)
}

func TestQobuzCredentialsShellFailure(t *testing.T) {
	defer mockey.UnPatchAll()
	qobuzCachedID = ""
	qobuzCachedSecret = ""
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Do")).Return(nil, errors.New("ko")).Build()

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsShellNonOK(t *testing.T) {
	defer mockey.UnPatchAll()
	qobuzCachedID = ""
	qobuzCachedSecret = ""
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Do")).Return(&http.Response{
		StatusCode: 503,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil).Build()

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsNoBundleScript(t *testing.T) {
	defer mockey.UnPatchAll()
	qobuzCachedID = ""
	qobuzCachedSecret = ""
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Do")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`<html>no script here</html>`)),
	}, nil).Build()

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsBundleFailure(t *testing.T) {
	defer mockey.UnPatchAll()
	qobuzCachedID = ""
	qobuzCachedSecret = ""
	callCount := 0
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Do")).To(func(_ *http.Request) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`<script src="/resources/1.0/js/main.js"></script>`)),
			}, nil
		}
		return nil, errors.New("bundle down")
	}).Build()

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsBundleNonOK(t *testing.T) {
	defer mockey.UnPatchAll()
	qobuzCachedID = ""
	qobuzCachedSecret = ""
	callCount := 0
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Do")).To(func(_ *http.Request) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`<script src="/resources/1.0/js/main.js"></script>`)),
			}, nil
		}
		return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader(""))}, nil
	}).Build()

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsNoCredentialsInBundle(t *testing.T) {
	defer mockey.UnPatchAll()
	qobuzCachedID = ""
	qobuzCachedSecret = ""
	callCount := 0
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Do")).To(func(_ *http.Request) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`<script src="/resources/1.0/js/main.js"></script>`)),
			}, nil
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`no credentials here`)),
		}, nil
	}).Build()

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsShellRequestBuildFailure(t *testing.T) {
	defer mockey.UnPatchAll()
	qobuzCachedID = ""
	qobuzCachedSecret = ""
	mockey.Mock(http.NewRequest).Return(nil, errors.New("ko")).Build()

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsShellReadFailure(t *testing.T) {
	defer mockey.UnPatchAll()
	qobuzCachedID = ""
	qobuzCachedSecret = ""
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Do")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil).Build()
	mockey.Mock(io.ReadAll).Return(nil, errors.New("ko")).Build()

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsBundleRequestBuildFailure(t *testing.T) {
	defer mockey.UnPatchAll()
	qobuzCachedID = ""
	qobuzCachedSecret = ""
	// control char in bundle URL causes http.NewRequest to fail
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Do")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("<script src=\"https://open.qobuz.com/\x00/js/main.js\"></script>")),
	}, nil).Build()

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCredentialsBundleReadFailure(t *testing.T) {
	defer mockey.UnPatchAll()
	qobuzCachedID = ""
	qobuzCachedSecret = ""
	callCount := 0
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Do")).To(func(_ *http.Request) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`<script src="/resources/1.0/js/main.js"></script>`))}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	}).Build()
	readCount := 0
	mockey.Mock(io.ReadAll).To(func(_ io.Reader) ([]byte, error) {
		readCount++
		if readCount == 1 {
			return []byte(`<script src="/resources/1.0/js/main.js"></script>`), nil
		}
		return nil, errors.New("ko")
	}).Build()

	_, _, err := qobuzCredentials()
	assert.NotNil(t, err)
}

func TestQobuzCDNURL(t *testing.T) {
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"url":"https://cdn.example/track.mp3"}`)),
	}, nil).Build()

	url, err := qobuzCDNURL("138731318")
	assert.Nil(t, err)
	assert.Equal(t, "https://cdn.example/track.mp3", url)
}

func TestQobuzCDNURLAllFailed(t *testing.T) {
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(http.DefaultClient, "Get")).Return(nil, errors.New("ko")).Build()

	url, err := qobuzCDNURL("138731318")
	assert.NotNil(t, err)
	assert.Empty(t, url)
}
