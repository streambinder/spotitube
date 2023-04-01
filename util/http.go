package util

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// this wouldn't be necessary but for testing
// as monkey patching HTTP library is definitely harder
// than doing for a custom and reusable portion of code
func HttpRequest(method, url string, queryParameters url.Values, body io.Reader, headers ...string) (*http.Response, error) {
	request, err := http.NewRequest(method, fmt.Sprintf("%s?%s", url, queryParameters.Encode()), body)
	if err != nil {
		return nil, err
	}

	for _, header := range headers {
		headerKeyValue := strings.SplitN(header, ":", 2)
		request.Header.Set(headerKeyValue[0], headerKeyValue[1])
	}
	return http.DefaultClient.Do(request)
}
