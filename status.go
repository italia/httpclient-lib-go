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

const maxRetryWait = 30 * time.Second

func parseRetryAfter(value string, now time.Time) (time.Duration, error) {
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err == nil {
		if seconds < 0 {
			return 0, fmt.Errorf("negative %s value %q", headerRetryAfter, value)
		}
		if seconds >= int64(maxRetryWait/time.Second) {
			return maxRetryWait, nil
		}

		return time.Duration(seconds) * time.Second, nil
	}

	retryAt, err := http.ParseTime(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s value %q", headerRetryAfter, value)
	}
	if !retryAt.After(now) {
		return 0, nil
	}

	return capRetryWait(retryAt.Sub(now)), nil
}

func parseRateLimitReset(value string, now time.Time) (time.Duration, error) {
	reset, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s value %q", headerRateReset, value)
	}

	waitSeconds := reset - now.Unix()
	if waitSeconds <= 0 {
		return 0, nil
	}
	if waitSeconds >= int64(maxRetryWait/time.Second) {
		return maxRetryWait, nil
	}

	return time.Duration(waitSeconds) * time.Second, nil
}

func capRetryWait(wait time.Duration) time.Duration {
	if wait > maxRetryWait {
		return maxRetryWait
	}

	return wait
}

func sleepWithContext(ctx context.Context, wait time.Duration) error {
	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

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

// statusTooManyRequests returns an HTTPResponse with the data from response.
func statusTooManyRequests(ctx context.Context, resp *http.Response, expBackoffAttempts int) (int, error) {
	// If Retry-after Header is set, use the header value.
	if retryAfter := resp.Header.Get(headerRetryAfter); retryAfter != "" {
		wait, err := parseRetryAfter(retryAfter, time.Now())
		if err == nil {
			log.Infof("Waiting: %s. (The value of %s)", wait, headerRetryAfter)
			if err = sleepWithContext(ctx, wait); err != nil {
				return expBackoffAttempts, err
			}

			return expBackoffAttempts, nil
		}
		log.Warn(err)
	}

	// Calculate ExpBackoff
	expBackoffWait := expBackoffCalc(expBackoffAttempts)
	// Perform a backoff sleep time.
	sleep := capRetryWait(time.Duration(expBackoffWait) * time.Second)
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
		wait, err := parseRetryAfter(retryAfter, time.Now())
		if err != nil {
			return expBackoffAttempts, err
		}
		log.Infof("Waiting: %s. (The value of %s)", wait, headerRetryAfter)
		if err = sleepWithContext(ctx, wait); err != nil {
			return expBackoffAttempts, err
		}
		return expBackoffAttempts, nil
	}

	// If X-rateLimit-remaining
	if reset := resp.Header.Get(headerRateReset); reset != "" {
		// If X-RateLimit-Remaining is set
		if remaining := resp.Header.Get(headerRateRemaining); remaining != "" {
			rateRemaining, err := strconv.Atoi(remaining)
			if err != nil {
				return expBackoffAttempts, err
			}
			if rateRemaining != 0 {
				// In this case there is another StatusForbidden and i should skip.
				return expBackoffAttempts, fmt.Errorf("forbidden resource")
			}

			wait, err := parseRateLimitReset(reset, time.Now())
			if err != nil {
				return expBackoffAttempts, err
			}
			log.Infof("Waiting %s for %s. (The difference between header %s and time.Now())", wait, headerRateReset, reset)
			if err = sleepWithContext(ctx, wait); err != nil {
				return expBackoffAttempts, err
			}
			return expBackoffAttempts, nil
		}
	}

	// Generic forbidden.
	return expBackoffAttempts, fmt.Errorf("forbidden resource")
}
