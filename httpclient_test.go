package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockDoer is a test implementation of HTTPDoer.
type mockDoer struct {
	resp *http.Response
	err  error
	got  *http.Request
}

func (m *mockDoer) Do(req *http.Request) (*http.Response, error) {
	m.got = req
	return m.resp, m.err
}

// handlerOneRepoList print on ResponseWriter one element of response list.
func handlerOneRepoList(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "{\"data\": \"example data\"}")
}

// handlerOneRepoListWithDelay print on ResponseWriter one element of response.
// with delay of 60 seconds.
func handlerOneRepoListWithDelay(w http.ResponseWriter, _ *http.Request) {
	time.Sleep(60 * time.Second)
	fmt.Fprint(w, "{\"data\": \"example data\"}")
}

// handlerEmpty print an empty response
func handlerEmpty(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "")
}

// handlerHeaderInResponse print something and return a specific Header.
func handlerHeaderInResponse(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("X-PowOfTwo", "4")
	w.Header().Write(w)

	fmt.Fprint(w, "{\"data\": \"example data\"}")
}

// TestGetUrl should test if a getUrl to a valid http resource will end without errors.
func TestGetUrl(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(handlerOneRepoList))
	defer ts.Close()

	resp, _ := GetURL(ts.URL, nil)
	r := "{\"data\": \"example data\"}"
	if string(resp.Body) != r && resp.Status.Code == 200 {
		t.Errorf("Call was incorrect, got: %s, want: %v.", resp.Body, r)
	}
}

// TestIncorrectProtocolUrl should test if a getUrl to incorrect protocol url will fail.
func TestIncorrectProtocolUrl(t *testing.T) {
	resp, err := GetURL("hktp://incorrectprotocol.url", nil)

	if err == nil && resp.Status.Code != -1 {
		t.Errorf("TestInexistentUrlCall was incorrect, got error: %v", err)
	}
}

// TestInexistentUrl should test if a getUrl to inexistent url will fail.
func TestInexistentUrl(t *testing.T) {
	resp, err := GetURL("http://inexistent.url", nil)
	if err == nil && resp.Status.Code == -1 {
		t.Errorf("TestInexistentUrlCall was incorrect, got error: %v", err)
	}
}

// TestEmptyResponse should test if a getUrl to a valid http resource with empty response will
// end without errors. Not really useful for getUrl(), use it as example.
func TestEmptyResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(handlerEmpty))
	defer ts.Close()

	resp, _ := GetURL(ts.URL, nil)
	r := ""
	if string(resp.Body) != r {
		t.Errorf("TestEmptyCall was incorrect, got: %s, want: %v.", string(resp.Body), r)
	}
}

// TestGetUrlWithDelayResponse getUrl test with 10 seconds response time.
func TestGetUrlWithDelayResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(handlerOneRepoListWithDelay))
	defer ts.Close()

	resp, _ := GetURL(ts.URL, nil)
	if resp.Body != nil {
		t.Errorf("TestCallWithDelay was incorrect, got: %s, want: %v.", string(resp.Body), nil)
	}
}

// TestGetUrlWithHeadersResponse getUrl test with headers in return.
func TestGetUrlWithHeadersResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(handlerHeaderInResponse))
	defer ts.Close()

	resp, _ := GetURL(ts.URL, nil)
	if resp.Headers.Get("X-PowOfTwo") != "4" {
		t.Errorf("TestCallWithDelay was incorrect, got: %s, want: %s.", resp.Headers.Get("X-PowOfTwo"), "4")
	}
}

// TestPostURL should test if a getUrl to a valid http resource will end without errors.
func TestPostURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(handlerOneRepoList))
	defer ts.Close()

	resp, _ := PostURL(ts.URL, nil, nil)
	r := "{\"data\": \"example data\"}"
	if string(resp.Body) != r && resp.Status.Code == 200 {
		t.Errorf("Call was incorrect, got: %s, want: %v.", resp.Body, r)
	}
}

// TestIncorrectProtocolUrl should test if a getUrl to incorrect protocol url will fail.
func TestPostIncorrectProtocolUrl(t *testing.T) {
	resp, err := PostURL("hktp://incorrectprotocol.url", nil, nil)

	if err == nil && resp.Status.Code != -1 {
		t.Errorf("TestInexistentUrlCall was incorrect, got error: %v", err)
	}
}

// TestInexistentUrl should test if a getUrl to inexistent url will fail.
func TestPostInexistentUrl(t *testing.T) {
	resp, err := PostURL("http://inexistent.url", nil, nil)
	if err == nil && resp.Status.Code == -1 {
		t.Errorf("TestInexistentUrlCall was incorrect, got error: %v", err)
	}
}

// TestEmptyResponse should test if a getUrl to a valid http resource with empty response will
// end without errors. Not really useful for getUrl(), use it as example.
func TestPostEmptyResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(handlerEmpty))
	defer ts.Close()

	resp, _ := PostURL(ts.URL, nil, nil)
	r := ""
	if string(resp.Body) != r {
		t.Errorf("TestEmptyCall was incorrect, got: %s, want: %v.", string(resp.Body), r)
	}
}

// TestPostUrlWithDelayResponse getUrl test with 10 seconds response time.
func TestPostUrlWithDelayResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(handlerOneRepoListWithDelay))
	defer ts.Close()

	resp, _ := PostURL(ts.URL, nil, nil)
	if resp.Body != nil {
		t.Errorf("TestCallWithDelay was incorrect, got: %s, want: %v.", string(resp.Body), nil)
	}
}

// TestPostUrlWithHeadersResponse getUrl test with headers in return.
func TestPostUrlWithHeadersResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(handlerHeaderInResponse))
	defer ts.Close()

	resp, _ := PostURL(ts.URL, nil, nil)
	if resp.Headers.Get("X-PowOfTwo") != "4" {
		t.Errorf("TestCallWithDelay was incorrect, got: %s, want: %s.", resp.Headers.Get("X-PowOfTwo"), "4")
	}
}

// TestNewClientNil verifies that NewClient(nil) returns a working client.
func TestNewClientNil(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(handlerOneRepoList))
	defer ts.Close()

	client := NewClient(nil)
	resp, err := client.GetURL(ts.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status.Code != 200 {
		t.Errorf("got status %d, want 200", resp.Status.Code)
	}
}

// TestNewClientCustomDoer verifies that a custom HTTPDoer is actually used.
func TestNewClientCustomDoer(t *testing.T) {
	rec := httptest.NewRecorder()
	fmt.Fprint(rec, `{"injected":true}`)
	doer := &mockDoer{resp: &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Header:     http.Header{"X-Custom": []string{"yes"}},
		Body:       rec.Result().Body,
	}}

	client := NewClient(doer)
	resp, err := client.GetURL("http://example.com", map[string]string{"Authorization": "Bearer token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doer.got == nil {
		t.Fatal("custom doer was not called")
	}
	if doer.got.Header.Get("Authorization") != "Bearer token" {
		t.Errorf("header not forwarded, got: %s", doer.got.Header.Get("Authorization"))
	}
	if resp.Headers.Get("X-Custom") != "yes" {
		t.Errorf("response header not returned, got: %s", resp.Headers.Get("X-Custom"))
	}
}

// TestClientGetURL tests Client.GetURL against a real test server.
func TestClientGetURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(handlerOneRepoList))
	defer ts.Close()

	client := NewClient(nil)
	resp, _ := client.GetURL(ts.URL, nil)
	r := "{\"data\": \"example data\"}"
	if string(resp.Body) != r && resp.Status.Code == 200 {
		t.Errorf("got: %s, want: %v.", resp.Body, r)
	}
}

// TestClientPostURL tests Client.PostURL against a real test server.
func TestClientPostURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(handlerOneRepoList))
	defer ts.Close()

	client := NewClient(nil)
	resp, _ := client.PostURL(ts.URL, nil, nil)
	r := "{\"data\": \"example data\"}"
	if string(resp.Body) != r && resp.Status.Code == 200 {
		t.Errorf("got: %s, want: %v.", resp.Body, r)
	}
}

// TestRequestWithContextAlreadyCancelled verifies that a pre-cancelled context returns immediately.
func TestRequestWithContextAlreadyCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(handlerOneRepoList))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewClient(nil)
	_, err := client.RequestWithContext(ctx, ts.URL, "GET", nil, nil)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

// TestRequestWithContextCancelInterruptsBackoff verifies that an explicit
// context cancellation does interrupt a Retry-After sleep.
func TestRequestWithContextCancelInterruptsBackoff(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	client := NewClient(nil)
	_, err := client.RequestWithContext(ctx, ts.URL, "GET", nil, nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if elapsed >= 5*time.Second {
		t.Errorf("cancel did not interrupt backoff: elapsed %v", elapsed)
	}
}
