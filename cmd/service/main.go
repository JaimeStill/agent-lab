package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/JaimeStill/agent-lab/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config load failed:", err)
	}

	if err := cfg.Finalize(); err != nil {
		log.Fatal("config finalize failed:", err)
	}

	svc, err := NewService(cfg)
	if err != nil {
		log.Fatal("service init failed:", err)
	}

	if err := svc.Start(); err != nil {
		log.Fatal("service start failed:", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

	if err := svc.Shutdown(cfg.ShutdownTimeoutDuration()); err != nil {
		log.Fatal("shutdown failed:", err)
	}

	log.Println("service stopped gracefully")
}
