package httpclient

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tomnomnom/linkheader"
)

// HTTPDoer is the interface for the underlying HTTP client. *http.Client satisfies it.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client wraps an HTTPDoer and exposes GetURL, PostURL and Request methods.
type Client struct {
	httpDoer HTTPDoer
}

// NewClient returns a Client that uses doer as the underlying HTTP client.
// If doer is nil, a default *http.Client with a 60-second timeout is used.
func NewClient(doer HTTPDoer) *Client {
	if doer == nil {
		doer = &http.Client{Timeout: 60 * time.Second}
	}

	return &Client{httpDoer: doer}
}

// defaultClient is used by the package-level functions for backwards compatibility.
//
//nolint:gochecknoglobals
var defaultClient = NewClient(nil)

// HTTPResponse wraps body, Status and Headers from the http.Response.
type HTTPResponse struct {
	Body    []byte
	Status  ResponseStatus
	Headers http.Header
}

// GetURL retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
func (c *Client) GetURL(url string, headers map[string]string) (HTTPResponse, error) {
	return c.Request(url, "GET", headers, nil)
}

// PostURL retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
func (c *Client) PostURL(url string, headers map[string]string, body io.Reader) (HTTPResponse, error) {
	return c.Request(url, "POST", headers, body)
}

// Request retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
func (c *Client) Request(url, verb string, headers map[string]string, body io.Reader) (HTTPResponse, error) {
	const maxBackOffAttempts = 8 // 2 minutes.

	attempts := 0

	var err error

	for attempts < maxBackOffAttempts {
		req, reqErr := http.NewRequestWithContext(context.Background(), verb, url, body)
		if reqErr != nil {
			return errResponse(reqErr, url), fmt.Errorf("build request: %w", reqErr)
		}

		for k, v := range headers {
			req.Header.Add(k, v)
		}

		resp, doErr := c.httpDoer.Do(req)
		if doErr != nil {
			return errResponse(doErr, url), fmt.Errorf("perform request: %w", doErr)
		}

		if resp.Body != nil {
			defer resp.Body.Close()
		}

		var httpResp HTTPResponse

		var done bool

		httpResp, done, attempts, err = evaluateResponse(resp, url, attempts)
		if done {
			return httpResp, err
		}
	}

	return HTTPResponse{
		Body:    nil,
		Status:  ResponseStatus{Text: "Invalid Status Code: " + url, Code: -1},
		Headers: nil,
	}, err
}

// evaluateResponse inspects resp and decides whether Request should return
// immediately (done == true) or retry with the returned attempts count.
func evaluateResponse(resp *http.Response, url string, attempts int) (HTTPResponse, bool, int, error) {
	switch {
	// Check if the request results in http OK.
	case resp.StatusCode >= http.StatusOK && resp.StatusCode <= 299:
		httpResp, err := statusOK(resp)

		return httpResp, true, attempts, err

	// Check if the request results in http notFound.
	case resp.StatusCode == http.StatusNotFound:
		log.Debugf("Status: %s - Resource: %s", resp.Status, url)

		httpResp, err := statusNotFound(resp)

		return httpResp, true, attempts, err

	// Check if the request results in http RateLimit error.
	case resp.StatusCode == http.StatusTooManyRequests:
		log.Debugf("Status: %s - Resource: %s", resp.Status, url)

		attempts = statusTooManyRequests(resp, attempts)

	// Check if the request result in http Forbidden status.
	case resp.StatusCode == http.StatusForbidden:
		log.Debugf("Status: %s - Resource: %s", resp.Status, url)

		next, err := statusForbidden(resp, attempts)
		if err != nil {
			return errResponse(err, url), true, attempts, err
		}

		attempts = next
	}

	return HTTPResponse{}, false, attempts + 1, nil
}

// errResponse builds the HTTPResponse returned alongside a transport level error.
func errResponse(err error, url string) HTTPResponse {
	return HTTPResponse{
		Body:    nil,
		Status:  ResponseStatus{Text: err.Error() + url, Code: -1},
		Headers: nil,
	}
}

// GetURL retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
func GetURL(url string, headers map[string]string) (HTTPResponse, error) {
	return defaultClient.Request(url, "GET", headers, nil)
}

// PostURL retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
func PostURL(url string, headers map[string]string, body io.Reader) (HTTPResponse, error) {
	return defaultClient.Request(url, "POST", headers, body)
}

// Request retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
func Request(url, verb string, headers map[string]string, body io.Reader) (HTTPResponse, error) {
	return defaultClient.Request(url, verb, headers, body)
}

// HeaderLink parse the Github Header Link to "next"/"last"/"first"/"prev" link of repositories.
// Example: HeaderLink(link,"next") or HeaderLink(link, "prev") or HeaderLink(link,"last").
func HeaderLink(linkHeader, command string) string {
	parsedLinks := linkheader.Parse(linkHeader)

	for _, link := range parsedLinks {
		if link.Rel == command {
			return link.URL
		}
	}

	return ""
}

// expBackoffCalc calculate the exponential backoff given.
func expBackoffCalc(attempts int) float64 {
	return (math.Pow(2, float64(attempts)) - 1) / 2
}
