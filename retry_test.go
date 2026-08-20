package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		value string
		want  time.Duration
		ok    bool
	}{
		{name: "seconds", value: "5", want: 5 * time.Second, ok: true},
		{name: "http date", value: now.Add(5 * time.Second).Format(http.TimeFormat), want: 5 * time.Second, ok: true},
		{name: "capped", value: "86400", want: maxRetryWait, ok: true},
		{name: "negative", value: "-1", ok: false},
		{name: "invalid", value: "invalid", ok: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseRetryAfter(test.value, now)
			if (err == nil) != test.ok {
				t.Fatalf("parseRetryAfter(%q) error = %v, want success = %v", test.value, err, test.ok)
			}
			if got != test.want {
				t.Fatalf("parseRetryAfter(%q) = %s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestParseRateLimitReset(t *testing.T) {
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		value string
		want  time.Duration
		ok    bool
	}{
		{name: "future", value: strconv.FormatInt(now.Add(5*time.Second).Unix(), 10), want: 5 * time.Second, ok: true},
		{name: "capped", value: strconv.FormatInt(now.Add(24*time.Hour).Unix(), 10), want: maxRetryWait, ok: true},
		{name: "past", value: strconv.FormatInt(now.Add(-time.Second).Unix(), 10), want: 0, ok: true},
		{name: "invalid", value: "invalid", ok: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseRateLimitReset(test.value, now)
			if (err == nil) != test.ok {
				t.Fatalf("parseRateLimitReset(%q) error = %v, want success = %v", test.value, err, test.ok)
			}
			if got != test.want {
				t.Fatalf("parseRateLimitReset(%q) = %s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestRequestWithContextDeadlineInterruptsRetryAfter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "86400")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := NewClient(nil).RequestWithContext(ctx, server.URL, http.MethodGet, nil, nil)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RequestWithContext() error = %v, want context deadline exceeded", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("RequestWithContext() took %s, want less than 500ms", elapsed)
	}
}
func TestRequestWithContextCancellationInterruptsRetryAfter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "86400")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	time.AfterFunc(50*time.Millisecond, cancel)

	start := time.Now()
	_, err := NewClient(nil).RequestWithContext(ctx, server.URL, http.MethodGet, nil, nil)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RequestWithContext() error = %v, want context canceled", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("RequestWithContext() took %s, want less than 500ms", elapsed)
	}
}
