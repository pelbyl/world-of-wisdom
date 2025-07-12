package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"world-of-wisdom/internal/server"
)

func main() {
	var (
		port       = flag.String("port", ":8080", "TCP port to listen on")
		difficulty = flag.Int("difficulty", 2, "Initial difficulty (1-6)")
		timeout    = flag.Duration("timeout", 30*time.Second, "Client timeout")
	)
	flag.Parse()

	cfg := server.Config{
		Port:       *port,
		Difficulty: *difficulty,
		Timeout:    *timeout,
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	if err := srv.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}