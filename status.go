package httpclient

import (
	"errors"
	"fmt"
	"io"
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
	headerRateReset     = "X-Ratelimit-Reset"
	headerRateRemaining = "X-Ratelimit-Remaining"
)

var (
	errNotFound  = errors.New("not found")
	errForbidden = errors.New("forbidden resource")
)

// statusOK returns an HTTPResponse with the data from response.
func statusOK(resp *http.Response) (HTTPResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err.Error())

		return HTTPResponse{
			Body:    nil,
			Status:  ResponseStatus{Text: resp.Status, Code: resp.StatusCode},
			Headers: resp.Header,
		}, fmt.Errorf("read response body: %w", err)
	}

	err = resp.Body.Close()
	if err != nil {
		log.Error(err.Error())
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
	}, errNotFound
}

// statusTooManyRequests sleeps according to the rate limit headers (or an
// exponential backoff when they are absent) and returns the updated attempt
// count.
func statusTooManyRequests(resp *http.Response, attempts int) int {
	if retryAfter := resp.Header.Get(headerRetryAfter); retryAfter != "" {
		log.Infof("Waiting: %s seconds. (The value of %s)", retryAfter, headerRetryAfter)

		secondsAfterRetry, err := strconv.Atoi(retryAfter)
		if err != nil {
			log.Warn(err)
		}

		time.Sleep(time.Second * time.Duration(secondsAfterRetry))

		return attempts
	}

	expBackoffWait := expBackoffCalc(attempts)
	sleep := time.Duration(expBackoffWait) * time.Second
	log.Infof("Rate limit reached, sleep %v \n", sleep)
	time.Sleep(sleep)

	return attempts + 1
}

// statusForbidden returns an HTTPResponse with the data from response.
func statusForbidden(resp *http.Response, expBackoffAttempts int) (int, error) {
	// If Retry-after is set, use that value.
	if retryAfter := resp.Header.Get(headerRetryAfter); retryAfter != "" {
		log.Infof("Waiting: %s seconds. (The value of %s)", retryAfter, headerRetryAfter)

		secondsAfterRetry, err := strconv.Atoi(retryAfter)
		if err != nil {
			log.Warn(err)
		}

		time.Sleep(time.Second * time.Duration(secondsAfterRetry))

		return expBackoffAttempts, nil
	}

	// If X-RateLimit-Reset and X-RateLimit-Remaining are both set.
	reset := resp.Header.Get(headerRateReset)
	remaining := resp.Header.Get(headerRateRemaining)

	if reset != "" && remaining != "" {
		rateRemaining, err := strconv.Atoi(remaining)
		if err != nil {
			log.Warn(err)
		}

		if rateRemaining != 0 {
			// In this case there is another StatusForbidden and i should skip.
			return expBackoffAttempts, errForbidden
		}

		retryEpoch, err := strconv.Atoi(reset)
		if err != nil {
			log.Warn(err)
		}

		secondsAfterRetry := int64(retryEpoch) - time.Now().Unix()
		log.Infof("Waiting %s seconds for %s. (The difference between header %s and time.Now())",
			strconv.FormatInt(secondsAfterRetry, 10), headerRateReset, reset)
		time.Sleep(time.Second * time.Duration(secondsAfterRetry))

		return expBackoffAttempts, nil
	}

	// Generic forbidden.
	return expBackoffAttempts, errForbidden
}
