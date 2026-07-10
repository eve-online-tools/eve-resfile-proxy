package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientRetriesOn5xx(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := &Client{
		HTTP:       server.Client(),
		Semaphore:  NewSemaphore(1),
		Timeout:    2 * time.Second,
		MaxRetries: -1,
	}
	body, err := client.GetBytes(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("GetBytes: %v", err)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q", body)
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d", attempts.Load())
	}
}

func TestClientNoRetryOn404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{
		HTTP:       server.Client(),
		Semaphore:  NewSemaphore(1),
		Timeout:    2 * time.Second,
		MaxRetries: -1,
	}
	if _, err := client.GetBytes(context.Background(), server.URL); err == nil {
		t.Fatal("expected error for 404")
	}
}
