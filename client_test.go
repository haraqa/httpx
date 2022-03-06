package httpx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/haraqa/httpx"
)

var (
	ErrInvalidCode = fmt.Errorf("invalid status code")
	ErrMissingBody = fmt.Errorf("missing body in response")
)

func DecorateWithContext(parent context.Context, client httpx.Client) httpx.ClientFunc {
	return func(r *http.Request) (*http.Response, error) {
		return client.Do(r.Clone(parent))
	}
}

func DecorateWithJSONBody(client httpx.Client, body interface{}) httpx.ClientFunc {
	return func(r *http.Request) (*http.Response, error) {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		r.Body = ioutil.NopCloser(bytes.NewReader(b))
		return client.Do(r)
	}
}

func DecorateWithHeader(client httpx.Client, key, value string) httpx.ClientFunc {
	return func(r *http.Request) (*http.Response, error) {
		r.Header.Add(key, value)
		return client.Do(r)
	}
}

func DecorateCheckStatus(client httpx.Client, code int) httpx.ClientFunc {
	return func(r *http.Request) (*http.Response, error) {
		resp, err := client.Do(r)
		if err != nil {
			return resp, err
		}
		if resp.StatusCode != code {
			return resp, fmt.Errorf("%w: expected %d, got %d", ErrInvalidCode, code, resp.StatusCode)
		}
		return resp, nil
	}
}

func DecorateCheckBody(client httpx.Client) httpx.ClientFunc {
	return func(r *http.Request) (*http.Response, error) {
		resp, err := client.Do(r)
		if err != nil {
			return resp, err
		}
		if resp.Body == nil {
			return resp, ErrMissingBody
		}
		return resp, nil
	}
}

func DecorateCheckHeaderError(client httpx.Client) httpx.ClientFunc {
	return func(r *http.Request) (*http.Response, error) {
		resp, err := client.Do(r)
		if err != nil {
			return resp, err
		}
		if errs := resp.Header.Get("ERRORS"); errs != "" {
			return resp, fmt.Errorf("received errors in header: %q", errs)
		}
		return resp, nil
	}
}

func DecorateDecodeJSON(client httpx.Client, dst interface{}) httpx.ClientFunc {
	return func(r *http.Request) (*http.Response, error) {
		resp, err := client.Do(r)
		if err != nil {
			return resp, err
		}
		err = json.NewDecoder(resp.Body).Decode(dst)
		return resp, err
	}
}

type MyType struct {
	Foo string `json:"foo"`
	Bar string `json:"bar"`
}

func PostValue(client httpx.Client, addr string, body interface{}) error {
	client = DecorateWithJSONBody(client, body)
	client = DecorateWithHeader(client, "X-CUSTOM-HEADER", "my-header")
	_, err := httpx.DoRequest(client, http.MethodPost, addr, nil)
	if err != nil {
		return err
	}

	return nil
}

func GetValue(client httpx.Client, addr string) (*MyType, error) {
	t := &MyType{}
	client = DecorateDecodeJSON(client, t)
	resp, err := httpx.DoRequest(client, http.MethodGet, addr, nil)
	_ = resp.Body.Close()
	return t, err
}

func ExampleClientFunc() {
	// to make our request we first set up our common http client
	var client httpx.Client
	client = http.DefaultClient
	client = DecorateCheckStatus(client, http.StatusOK)
	client = DecorateCheckHeaderError(client)
	client = DecorateCheckBody(client)
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*60)
	defer cancel()
	client = DecorateWithContext(ctx, client)

	// now we can call our PostValue function
	err := PostValue(client, "http://example.com/my/api", MyType{Foo: "bar"})
	if err != nil {
		panic(err)
	}
	// and a new call for our GetValue
	t, err := GetValue(client, "http://example.com/my/api?foo=bar")
	if err != nil {
		panic(err)
	}
	fmt.Println(t)
}
