package httpx

import (
	"context"
	"io"
	"net/http"
)

// ensure Client always matches with http.DefaultClient and ClientFunc
var (
	_ Client = http.DefaultClient
	_ Client = func() ClientFunc { return func(r *http.Request) (*http.Response, error) { return nil, nil } }()
)

// Client is an interface compatible with the *http.Client type
type Client interface {
	Do(*http.Request) (*http.Response, error)
}

// ClientFunc is an adapter implmenented by Client and can be used to create new Clients using just a function
type ClientFunc func(*http.Request) (*http.Response, error)

// Do calls the parent function
func (c ClientFunc) Do(req *http.Request) (*http.Response, error) {
	return c(req)
}

// DoRequestWithContext wraps http.NewRequestWithContext and uses the returned request in a client.Do call
func DoRequestWithContext(ctx context.Context, client Client, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

// DoRequest wraps DoRequestWithContext using context.Background.
func DoRequest(client Client, method, url string, body io.Reader) (*http.Response, error) {
	return DoRequestWithContext(context.Background(), client, method, url, body)
}
