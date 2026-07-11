package cdn

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"
)

const DefaultUserAgent = "github.com/eve-online-tools/eve-resfile-proxy"

const (
	defaultMaxRetries  = 3
	defaultConcurrency = 4
	initialBackoff     = 250 * time.Millisecond
	maxRetryAfter      = 30 * time.Second
)

// UserAgentRoundTripper sets User-Agent on outgoing requests.
type UserAgentRoundTripper struct {
	Base      http.RoundTripper
	UserAgent string
}

func (t UserAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.UserAgent == "" {
		return roundTripperBase(t.Base).RoundTrip(req)
	}

	cloned := req.Clone(req.Context())
	cloned.Header.Set("User-Agent", t.UserAgent)
	return roundTripperBase(t.Base).RoundTrip(cloned)
}

// RetryRoundTripper retries round trips that fail with transport errors or
// retryable HTTP status codes, using exponential backoff.
type RetryRoundTripper struct {
	Base       http.RoundTripper
	MaxRetries int
}

func (t RetryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	retries := t.maxRetriesOrDefault()
	base := roundTripperBase(t.Base)

	for attempt := 0; ; attempt++ {
		resp, err := base.RoundTrip(req)
		if err != nil {
			if attempt >= retries {
				return nil, err
			}
			if err := sleep(req.Context(), backoff(attempt)); err != nil {
				return nil, err
			}
			continue
		}

		if !isRetryableStatus(resp.StatusCode) || attempt >= retries {
			return resp, nil
		}

		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		wait := backoff(attempt)
		if resp.StatusCode == http.StatusTooManyRequests && retryAfter > 0 {
			wait = maxDuration(wait, retryAfter)
		}
		if err := sleep(req.Context(), wait); err != nil {
			return nil, err
		}
	}
}

// ConcurrencyRoundTripper limits concurrent in-flight requests.
type ConcurrencyRoundTripper struct {
	Base http.RoundTripper
	sem  chan struct{}
}

// NewConcurrencyRoundTripper returns a RoundTripper that allows at most limit
// concurrent requests. Values below 1 use defaultConcurrency.
func NewConcurrencyRoundTripper(base http.RoundTripper, limit int) *ConcurrencyRoundTripper {
	if limit < 1 {
		limit = defaultConcurrency
	}
	return &ConcurrencyRoundTripper{
		Base: base,
		sem:  make(chan struct{}, limit),
	}
}

func (t *ConcurrencyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	select {
	case t.sem <- struct{}{}:
		defer func() { <-t.sem }()
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}
	return roundTripperBase(t.Base).RoundTrip(req)
}

func configureHTTPClient(client *http.Client, o options) *http.Client {
	c := *client

	rt := roundTripperBase(c.Transport)
	userAgent := o.userAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}
	rt = UserAgentRoundTripper{Base: rt, UserAgent: userAgent}
	if !o.withoutRetries {
		rt = RetryRoundTripper{Base: rt, MaxRetries: defaultMaxRetries}
	}
	if limit, ok := o.concurrencyLimit(); ok {
		rt = NewConcurrencyRoundTripper(rt, limit)
	}
	c.Transport = rt

	return &c
}

func roundTripperBase(rt http.RoundTripper) http.RoundTripper {
	if rt != nil {
		return rt
	}
	return http.DefaultTransport
}

func (t RetryRoundTripper) maxRetriesOrDefault() int {
	if t.MaxRetries > 0 {
		return t.MaxRetries
	}
	return defaultMaxRetries
}

func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func backoff(attempt int) time.Duration {
	return initialBackoff * time.Duration(1<<attempt)
}

func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(header); err == nil {
		d := time.Duration(seconds) * time.Second
		if d > maxRetryAfter {
			return maxRetryAfter
		}
		return d
	}
	if t, err := http.ParseTime(header); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		if d > maxRetryAfter {
			return maxRetryAfter
		}
		return d
	}
	return 0
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func (o options) concurrencyLimit() (limit int, enabled bool) {
	if o.concurrency == nil {
		return defaultConcurrency, true
	}
	if *o.concurrency == 0 {
		return 0, false
	}
	return *o.concurrency, true
}
