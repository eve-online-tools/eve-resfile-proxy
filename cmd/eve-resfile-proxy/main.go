package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eve-online-tools/eve-resfile-proxy/service"
	"github.com/eve-online-tools/eve-resfile-proxy/service/assetcache"
)

func main() {
	cfg, err := parseConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	var opts []service.Option
	if cfg.CacheDir != "" {
		opts = append(opts, service.WithCache(assetcache.New(cfg.CacheDir)))
	}

	svc, err := service.New(context.Background(), cfg.serviceConfig(), opts...)
	if err != nil {
		log.Fatalf("service: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := svc.Start(ctx); err != nil {
		log.Fatalf("start: %v", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := svc.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
