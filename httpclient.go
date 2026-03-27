package httpclient

import (
	"context"
	"io"
	"math"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tomnomnom/linkheader"
)

// HTTPDoer is the interface for the underlying HTTP client. *http.Client satisfies it.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
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

// defaultClient is used by the package-level functions for backward compatibility.
var defaultClient = NewClient(nil)

// HTTPResponse wraps body, Status and Headers from the http.Response.
type HTTPResponse struct {
	Body    []byte
	Status  ResponseStatus
	Headers http.Header
}

// GetURL retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
func (c *Client) GetURL(URL string, headers map[string]string) (HTTPResponse, error) {
	return c.RequestWithContext(context.Background(), URL, "GET", headers, nil)
}

// GetURLWithContext retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
// Backoff sleeps are interrupted if ctx is cancelled.
func (c *Client) GetURLWithContext(ctx context.Context, URL string, headers map[string]string) (HTTPResponse, error) {
	return c.RequestWithContext(ctx, URL, "GET", headers, nil)
}

// PostURL retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
func (c *Client) PostURL(URL string, headers map[string]string, body io.Reader) (HTTPResponse, error) {
	return c.RequestWithContext(context.Background(), URL, "POST", headers, body)
}

// PostURLWithContext retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
// Backoff sleeps are interrupted if ctx is cancelled.
func (c *Client) PostURLWithContext(ctx context.Context, URL string, headers map[string]string, body io.Reader) (HTTPResponse, error) {
	return c.RequestWithContext(ctx, URL, "POST", headers, body)
}

// Request retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
func (c *Client) Request(URL string, verb string, headers map[string]string, body io.Reader) (HTTPResponse, error) {
	return c.RequestWithContext(context.Background(), URL, verb, headers, body)
}

// RequestWithContext retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
// Backoff sleeps are interrupted if ctx is cancelled.
func (c *Client) RequestWithContext(ctx context.Context, URL string, verb string, headers map[string]string, body io.Reader) (HTTPResponse, error) {
	expBackoffAttempts := 0
	const maxBackOffAttempts = 8 // 2 minutes.
	var err error

	for expBackoffAttempts < maxBackOffAttempts {

		req, err := http.NewRequestWithContext(ctx, verb, URL, body)
		if err != nil {
			return HTTPResponse{
				Body:    nil,
				Status:  ResponseStatus{Text: err.Error() + URL, Code: -1},
				Headers: nil,
			}, err
		}

		// Set headers.
		for k, v := range headers {
			req.Header.Add(k, v)
		}

		// Perform the request.
		resp, err := c.httpDoer.Do(req)
		if err != nil {
			return HTTPResponse{
				Body:    nil,
				Status:  ResponseStatus{Text: err.Error() + URL, Code: -1},
				Headers: nil,
			}, err
		}

		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
		}

		// Check if the request results in http OK.
		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			return statusOK(resp)
		}

		// Check if the request results in http notFound.
		if resp.StatusCode == http.StatusNotFound {
			log.Debugf("Status: %s - Resource: %s", resp.Status, URL)
			return statusNotFound(resp)
		}

		// Check if the request results in http RateLimit error.
		if resp.StatusCode == http.StatusTooManyRequests {
			log.Debugf("Status: %s - Resource: %s", resp.Status, URL)
			expBackoffAttempts, err = statusTooManyRequests(ctx, resp, expBackoffAttempts)
			if err != nil {
				return HTTPResponse{
					Body:    nil,
					Status:  ResponseStatus{Text: err.Error() + URL, Code: -1},
					Headers: nil,
				}, err
			}

		}
		// Check if the request result in http Forbidden status.
		if resp.StatusCode == http.StatusForbidden {
			log.Debugf("Status: %s - Resource: %s", resp.Status, URL)
			expBackoffAttempts, err = statusForbidden(ctx, resp, expBackoffAttempts)
			if err != nil {
				return HTTPResponse{
					Body:    nil,
					Status:  ResponseStatus{Text: err.Error() + URL, Code: -1},
					Headers: nil,
				}, err
			}
		}

		expBackoffAttempts += 1
	}

	// Generic invalid status code.
	return HTTPResponse{
		Body:    nil,
		Status:  ResponseStatus{Text: "Invalid Status Code: " + URL, Code: -1},
		Headers: nil,
	}, err
}

// GetURL retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
func GetURL(URL string, headers map[string]string) (HTTPResponse, error) {
	return defaultClient.RequestWithContext(context.Background(), URL, "GET", headers, nil)
}

// GetURLWithContext retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
// Backoff sleeps are interrupted if ctx is cancelled.
func GetURLWithContext(ctx context.Context, URL string, headers map[string]string) (HTTPResponse, error) {
	return defaultClient.RequestWithContext(ctx, URL, "GET", headers, nil)
}

// PostURL retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
func PostURL(URL string, headers map[string]string, body io.Reader) (HTTPResponse, error) {
	return defaultClient.RequestWithContext(context.Background(), URL, "POST", headers, body)
}

// PostURLWithContext retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
// Backoff sleeps are interrupted if ctx is cancelled.
func PostURLWithContext(ctx context.Context, URL string, headers map[string]string, body io.Reader) (HTTPResponse, error) {
	return defaultClient.RequestWithContext(ctx, URL, "POST", headers, body)
}

// Request retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
func Request(URL string, verb string, headers map[string]string, body io.Reader) (HTTPResponse, error) {
	return defaultClient.RequestWithContext(context.Background(), URL, verb, headers, body)
}

// RequestWithContext retrieves data, status and response headers from an URL.
// It uses some technique to slow down the requests if it get a 429 (Too Many Requests) response.
// Backoff sleeps are interrupted if ctx is cancelled.
func RequestWithContext(ctx context.Context, URL string, verb string, headers map[string]string, body io.Reader) (HTTPResponse, error) {
	return defaultClient.RequestWithContext(ctx, URL, verb, headers, body)
}

// HeaderLink parse the GitHub Header Link to "next"/"last"/"first"/"prev" link of repositories.
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
