package cdn_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/common/domain"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
	"github.com/eve-online-tools/eve-resfile-proxy/vfs/fetch/cdn"
)

func TestUserAgentRoundTripper_setsHeader(t *testing.T) {
	var gotUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	client := server.Client()
	client.Transport = cdn.UserAgentRoundTripper{
		Base:      client.Transport,
		UserAgent: "test-agent/1.0",
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	_ = resp.Body.Close()

	if gotUA != "test-agent/1.0" {
		t.Fatalf("User-Agent = %q", gotUA)
	}
}

func TestRetryRoundTripper_retriesTransportError(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 2 {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("server does not support hijack")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatalf("Hijack: %v", err)
			}
			_ = conn.Close()
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	client := server.Client()
	client.Transport = cdn.RetryRoundTripper{Base: client.Transport, MaxRetries: 3}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	_ = resp.Body.Close()
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d", attempts.Load())
	}
}

func TestRetryRoundTripper_retriesRetryableStatus(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	client := server.Client()
	client.Transport = cdn.RetryRoundTripper{Base: client.Transport, MaxRetries: 3}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d", attempts.Load())
	}
}

func TestNew_appliesDefaultUserAgent(t *testing.T) {
	var gotUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	fetcher := cdn.New(server.Client(), cdn.WithDomain(domain.Domain(server.URL)))

	if _, err := fetcher.FetchEntry(context.Background(), vfs.Entry{CDNPath: "7d/icon"}); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if gotUA != cdn.DefaultUserAgent {
		t.Fatalf("User-Agent = %q, want %q", gotUA, cdn.DefaultUserAgent)
	}
}

func TestNew_withUserAgent(t *testing.T) {
	const wantUA = "custom-agent/2.0"
	var gotUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	fetcher := cdn.New(server.Client(), cdn.WithDomain(domain.Domain(server.URL)), cdn.WithUserAgent(wantUA))

	if _, err := fetcher.FetchEntry(context.Background(), vfs.Entry{CDNPath: "7d/icon"}); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if gotUA != wantUA {
		t.Fatalf("User-Agent = %q, want %q", gotUA, wantUA)
	}
}

func TestNew_retriesByDefault(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	fetcher := cdn.New(server.Client(), cdn.WithDomain(domain.Domain(server.URL)))

	data, err := fetcher.FetchEntry(context.Background(), vfs.Entry{CDNPath: "7d/icon"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(data) != "ok" {
		t.Fatalf("data = %q", data)
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d", attempts.Load())
	}
}

func TestNew_withoutRetries(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)

	fetcher := cdn.New(server.Client(), cdn.WithDomain(domain.Domain(server.URL)), cdn.WithoutRetries())

	if _, err := fetcher.FetchEntry(context.Background(), vfs.Entry{CDNPath: "7d/icon"}); err == nil {
		t.Fatal("expected fetch error")
	}
	if attempts.Load() != 1 {
		t.Fatalf("attempts = %d", attempts.Load())
	}
}

func TestConcurrencyRoundTripper_limitsInFlight(t *testing.T) {
	var inFlight atomic.Int32
	var maxInFlight atomic.Int32
	release := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := inFlight.Add(1)
		for {
			peak := maxInFlight.Load()
			if cur <= peak || maxInFlight.CompareAndSwap(peak, cur) {
				break
			}
		}
		<-release
		inFlight.Add(-1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	const limit = 2
	client := server.Client()
	client.Transport = cdn.NewConcurrencyRoundTripper(client.Transport, limit)

	var wg sync.WaitGroup
	for range limit * 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(server.URL)
			if err != nil {
				t.Errorf("Get: %v", err)
				return
			}
			_ = resp.Body.Close()
		}()
	}

	deadline := time.After(500 * time.Millisecond)
	for maxInFlight.Load() < limit {
		select {
		case <-deadline:
			t.Fatalf("max in-flight = %d, want %d", maxInFlight.Load(), limit)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	close(release)
	wg.Wait()
}

func TestNew_withConcurrencyZeroDisablesLimiter(t *testing.T) {
	var inFlight atomic.Int32
	var maxInFlight atomic.Int32
	release := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := inFlight.Add(1)
		for {
			peak := maxInFlight.Load()
			if cur <= peak || maxInFlight.CompareAndSwap(peak, cur) {
				break
			}
		}
		<-release
		inFlight.Add(-1)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	fetcher := cdn.New(server.Client(), cdn.WithDomain(domain.Domain(server.URL)), cdn.WithConcurrency(0))

	const workers = 8
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := fetcher.FetchEntry(context.Background(), vfs.Entry{CDNPath: "7d/icon"}); err != nil {
				t.Errorf("Fetch: %v", err)
			}
		}()
	}

	deadline := time.After(500 * time.Millisecond)
	for maxInFlight.Load() < workers {
		select {
		case <-deadline:
			t.Fatalf("max in-flight = %d, want at least %d without limiter", maxInFlight.Load(), workers)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	close(release)
	wg.Wait()
}

func TestNew_withConcurrency(t *testing.T) {
	var inFlight atomic.Int32
	var maxInFlight atomic.Int32
	release := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := inFlight.Add(1)
		for {
			peak := maxInFlight.Load()
			if cur <= peak || maxInFlight.CompareAndSwap(peak, cur) {
				break
			}
		}
		<-release
		inFlight.Add(-1)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	const limit = 2
	fetcher := cdn.New(server.Client(), cdn.WithDomain(domain.Domain(server.URL)), cdn.WithConcurrency(limit))

	var wg sync.WaitGroup
	for range limit * 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := fetcher.FetchEntry(context.Background(), vfs.Entry{CDNPath: "7d/icon"}); err != nil {
				t.Errorf("Fetch: %v", err)
			}
		}()
	}

	deadline := time.After(500 * time.Millisecond)
	for maxInFlight.Load() < limit {
		select {
		case <-deadline:
			t.Fatalf("max in-flight = %d, want %d", maxInFlight.Load(), limit)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	close(release)
	wg.Wait()

	if peak := maxInFlight.Load(); peak > limit {
		t.Fatalf("max in-flight = %d, want <= %d", peak, limit)
	}
}
