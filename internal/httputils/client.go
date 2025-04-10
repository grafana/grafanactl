package httputils

import (
	"net/http"
	"time"
)

func NewHTTPClient() (*http.Client, error) {
	return &http.Client{
		Timeout:   10 * time.Second, // TODO: make this configurable maybe?
		Transport: &LoggedHTTPRoundTripper{},
	}, nil
}
