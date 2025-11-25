package internal_server_test

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/server"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
}

func getAvailablePort(t *testing.T) int {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}

func TestNew(t *testing.T) {
	cfg := &config.ServerConfig{
		Host:            "localhost",
		Port:            8080,
		ReadTimeout:     "30s",
		WriteTimeout:    "30s",
		ShutdownTimeout: "30s",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	sys := server.New(cfg, handler, testLogger())

	if sys == nil {
		t.Fatal("New() returned nil")
	}
}

func TestStart_ServerResponds(t *testing.T) {
	port := getAvailablePort(t)

	cfg := &config.ServerConfig{
		Host:            "localhost",
		Port:            port,
		ReadTimeout:     "5s",
		WriteTimeout:    "5s",
		ShutdownTimeout: "5s",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	sys := server.New(cfg, handler, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	err := sys.Start(ctx, &wg)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/test", port))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	cancel()
	time.Sleep(200 * time.Millisecond)
}

func TestStop(t *testing.T) {
	port := getAvailablePort(t)

	cfg := &config.ServerConfig{
		Host:            "localhost",
		Port:            port,
		ReadTimeout:     "5s",
		WriteTimeout:    "5s",
		ShutdownTimeout: "5s",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	sys := server.New(cfg, handler, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	err := sys.Start(ctx, &wg)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	_, err = http.Get(fmt.Sprintf("http://localhost:%d/test", port))
	if err != nil {
		t.Fatalf("Server not responding before stop: %v", err)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	err = sys.Stop(stopCtx)
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	_, err = http.Get(fmt.Sprintf("http://localhost:%d/test", port))
	if err == nil {
		t.Error("Server still responding after stop")
	}
}

func TestGracefulShutdown(t *testing.T) {
	port := getAvailablePort(t)

	cfg := &config.ServerConfig{
		Host:            "localhost",
		Port:            port,
		ReadTimeout:     "5s",
		WriteTimeout:    "5s",
		ShutdownTimeout: "5s",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("completed"))
	})

	sys := server.New(cfg, handler, testLogger())

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	err := sys.Start(ctx, &wg)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	done := make(chan bool)
	go func() {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/test", port))
		if err != nil {
			t.Errorf("Request failed during shutdown: %v", err)
			done <- false
			return
		}
		defer resp.Body.Close()
		done <- resp.StatusCode == http.StatusOK
	}()

	time.Sleep(25 * time.Millisecond)
	cancel()

	success := <-done
	if !success {
		t.Error("In-flight request not completed during graceful shutdown")
	}

	time.Sleep(200 * time.Millisecond)
}
