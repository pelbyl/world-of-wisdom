package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const (
	defaultPort = ":8080"
)

func main() {
	port := defaultPort
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", port, err)
	}
	defer listener.Close()

	log.Printf("TCP server listening on %s", port)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Failed to accept connection: %v", err)
				continue
			}

			go handleConnection(conn)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	
	log.Printf("New connection from %s", conn.RemoteAddr())
	
	_, err := conn.Write([]byte("Connected to Word of Wisdom server\n"))
	if err != nil {
		log.Printf("Failed to write to connection: %v", err)
		return
	}
}