package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultMaxRetries   = 3
	initialBackoff      = 250 * time.Millisecond
	maxRetryAfter       = 30 * time.Second
	defaultFetchTimeout = 30 * time.Second
	defaultConcurrency  = 8
)

type Client struct {
	HTTP       *http.Client
	Semaphore  *Semaphore
	Timeout    time.Duration
	MaxRetries int // negative uses defaultMaxRetries
}

func NewClient() *Client {
	return NewFromClient(&http.Client{Timeout: defaultFetchTimeout})
}

func NewFromClient(client *http.Client) *Client {
	return &Client{
		HTTP:       client,
		Semaphore:  NewSemaphore(defaultConcurrency),
		Timeout:    defaultFetchTimeout,
		MaxRetries: -1,
	}
}

func (c *Client) GetBytes(ctx context.Context, url string) ([]byte, error) {
	return c.getBytes(ctx, url)
}

func (c *Client) GetText(ctx context.Context, url string) (string, error) {
	body, err := c.GetBytes(ctx, url)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *Client) getBytes(ctx context.Context, url string) ([]byte, error) {
	c.Semaphore.Acquire()
	defer c.Semaphore.Release()

	timeout := c.Timeout
	if timeout == 0 {
		timeout = defaultFetchTimeout
	}

	var statusHistory []int
	retries := defaultMaxRetries
	if c.MaxRetries >= 0 {
		retries = c.MaxRetries
	}
	maxAttempts := retries + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		body, status, retryAfter, err := c.doAttempt(reqCtx, url)
		cancel()
		if err != nil {
			if attempt == retries {
				return nil, fmt.Errorf("fetch %s after %d attempts: %w", url, maxAttempts, err)
			}
			if err := sleep(ctx, backoff(attempt)); err != nil {
				return nil, err
			}
			continue
		}
		if status == http.StatusOK {
			return body, nil
		}

		statusHistory = append(statusHistory, status)

		if status == http.StatusNotFound {
			return nil, fmt.Errorf("fetch %s: not found (HTTP 404)", url)
		}

		if attempt == retries {
			return nil, fmt.Errorf("fetch %s after %d attempts: HTTP %v", url, maxAttempts, statusHistory)
		}

		wait := backoff(attempt)
		if status == http.StatusTooManyRequests && retryAfter > 0 {
			wait = maxDuration(wait, retryAfter)
		}
		if err := sleep(ctx, wait); err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("fetch %s failed", url)
}

func (c *Client) doAttempt(ctx context.Context, url string) ([]byte, int, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, 0, err
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, resp.StatusCode, retryAfter, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, 0, err
	}
	return body, resp.StatusCode, 0, nil
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
