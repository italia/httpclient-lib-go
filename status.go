package httpclient

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

// ResponseStatus contains the status and statusCode of a response.
type ResponseStatus struct {
	Text string // e.g. "200 OK"
	Code int    // e.g. 200
}

const (
	headerRetryAfter    = "Retry-After"
	headerRateReset     = "X-RateLimit-Reset"
	headerRateRemaining = "X-RateLimit-Remaining"
)

// statusOK returns an HTTPResponse with the data from response.
func statusOK(resp *http.Response) (HTTPResponse, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf(err.Error())
		return HTTPResponse{
			Body:    nil,
			Status:  ResponseStatus{Text: resp.Status, Code: resp.StatusCode},
			Headers: resp.Header,
		}, err
	}

	err = resp.Body.Close()
	if err != nil {
		log.Errorf(err.Error())
	}

	return HTTPResponse{
		Body:    body,
		Status:  ResponseStatus{Text: resp.Status, Code: resp.StatusCode},
		Headers: resp.Header,
	}, nil
}

// statusNotFound returns an HTTPResponse with the data from response.
func statusNotFound(resp *http.Response) (HTTPResponse, error) {
	return HTTPResponse{
		Body:    nil,
		Status:  ResponseStatus{Text: resp.Status, Code: resp.StatusCode},
		Headers: resp.Header,
	}, fmt.Errorf("not found")
}

// sleepWithContext sleeps for d, but returns early if ctx is explicitly cancelled.
// A context deadline expiring does not interrupt the sleep: deadlines apply to
// individual HTTP requests (via http.Client.Timeout or http.NewRequestWithContext),
// not to retry waits dictated by the server via Retry-After.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	start := time.Now()
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		if ctx.Err() == context.Canceled {
			return ctx.Err()
		}
		// DeadlineExceeded: sleep for the remaining duration.
		if remaining := d - time.Since(start); remaining > 0 {
			time.Sleep(remaining)
		}
		return nil
	}
}

// statusTooManyRequests returns an HTTPResponse with the data from response.
func statusTooManyRequests(ctx context.Context, resp *http.Response, expBackoffAttempts int) (int, error) {
	// If Retry-after Header is set, use the header value.
	if retryAfter := resp.Header.Get(headerRetryAfter); retryAfter != "" {
		log.Infof("Waiting: %s seconds. (The value of %s)", retryAfter, headerRetryAfter)
		secondsAfterRetry, err := strconv.Atoi(retryAfter)
		if err != nil {
			log.Warn(err)
		}
		if err := sleepWithContext(ctx, time.Second*time.Duration(secondsAfterRetry)); err != nil {
			return expBackoffAttempts, err
		}
		return expBackoffAttempts, nil
	}
	// Calculate ExpBackoff
	expBackoffWait := expBackoffCalc(expBackoffAttempts)
	// Perform a backoff sleep time.
	sleep := time.Duration(expBackoffWait) * time.Second
	log.Infof("Rate limit reached, sleep %v \n", sleep)
	if err := sleepWithContext(ctx, sleep); err != nil {
		return expBackoffAttempts, err
	}

	return expBackoffAttempts + 1, nil
}

// statusForbidden returns an HTTPResponse with the data from response.
func statusForbidden(ctx context.Context, resp *http.Response, expBackoffAttempts int) (int, error) {
	// If Retry-after is set, use that value.
	if retryAfter := resp.Header.Get(headerRetryAfter); retryAfter != "" {
		log.Infof("Waiting: %s seconds. (The value of %s)", retryAfter, headerRetryAfter)
		secondsAfterRetry, err := strconv.Atoi(retryAfter)
		if err != nil {
			log.Warn(err)
		}
		if err := sleepWithContext(ctx, time.Second*time.Duration(secondsAfterRetry)); err != nil {
			return expBackoffAttempts, err
		}
		return expBackoffAttempts, nil
	}

	// If X-rateLimit-remaining
	if reset := resp.Header.Get(headerRateReset); reset != "" {
		// If X-RateLimit-Remaining is set
		if remaining := resp.Header.Get(headerRateRemaining); reset != "" {
			rateRemaining, err := strconv.Atoi(remaining)
			if err != nil {
				log.Warn(err)
			}
			if rateRemaining != 0 {
				// In this case there is another StatusForbidden and i should skip.
				return expBackoffAttempts, fmt.Errorf("forbidden resource")
			}

			retryEpoch, err := strconv.Atoi(reset)
			if err != nil {
				log.Warn(err)
			}
			secondsAfterRetry := int64(retryEpoch) - time.Now().Unix()
			log.Infof("Waiting %s seconds for %s. (The difference between header %s and time.Now())", strconv.FormatInt(secondsAfterRetry, 10), headerRateReset, reset)
			if err := sleepWithContext(ctx, time.Second*time.Duration(secondsAfterRetry)); err != nil {
				return expBackoffAttempts, err
			}
			return expBackoffAttempts, nil
		}
	}

	// Generic forbidden.
	return expBackoffAttempts, fmt.Errorf("forbidden resource")

}
