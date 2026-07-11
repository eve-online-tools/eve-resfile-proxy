package buildnumber_test

import (
	"sync"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/service/buildnumber"
)

func TestBuildNumberSetGet(t *testing.T) {
	t.Parallel()

	b := &buildnumber.BuildNumber{}
	if got := b.Get(); got != "" {
		t.Fatalf("Get() = %q, want empty", got)
	}

	b.Set("123456")
	if got := b.Get(); got != "123456" {
		t.Fatalf("Get() = %q, want 123456", got)
	}

	b.Set("789")
	if got := b.Get(); got != "789" {
		t.Fatalf("Get() = %q, want 789", got)
	}
}

func TestBuildNumberConcurrent(t *testing.T) {
	t.Parallel()

	b := &buildnumber.BuildNumber{}
	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				b.Set("build")
				_ = b.Get()
			}
		}(i)
	}

	wg.Wait()
}
