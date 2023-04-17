package util

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func TestHttpRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "response")
		}))
	defer server.Close()

	response, err := HttpRequest(context.Background(), http.MethodGet, server.URL, nil, nil, "Host:localhost")
	assert.Nil(t, err)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	assert.Nil(t, err)
	assert.Equal(t, []byte("response"), body)
}

func TestHttpRequestFailure(t *testing.T) {
	// monkey patching
	patchhttpNewRequestWithContext := gomonkey.ApplyFunc(http.NewRequestWithContext,
		func(context.Context, string, string, io.Reader) (*http.Request, error) {
			return nil, errors.New("failure")
		})
	defer patchhttpNewRequestWithContext.Reset()

	// testing
	assert.Error(t, ErrOnly(HttpRequest(context.Background(), http.MethodGet, "localhost", nil, nil)), "failure")
}
