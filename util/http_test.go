package util

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
)

func TestHttpRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "response")
		}))
	defer server.Close()

	response, err := HttpRequest(http.MethodGet, server.URL, nil, nil, "Host:localhost")
	assert.Nil(t, err)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	assert.Nil(t, err)
	assert.Equal(t, []byte("response"), body)
}

func TestHttpRequestFailure(t *testing.T) {
	// monkey patching
	monkey.Patch(http.NewRequestWithContext,
		func(context.Context, string, string, io.Reader) (*http.Request, error) {
			return nil, errors.New("failure")
		})
	defer monkey.Unpatch(http.NewRequestWithContext)

	// testing
	assert.Error(t, ErrOnly(HttpRequest(http.MethodGet, "localhost", nil, nil)), "failure")
}