package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

const (
	defaultServer = "localhost:8080"
)

func main() {
	server := defaultServer
	if len(os.Args) > 1 {
		server = os.Args[1]
	}

	conn, err := net.Dial("tcp", server)
	if err != nil {
		log.Fatalf("Failed to connect to server %s: %v", server, err)
	}
	defer conn.Close()

	log.Printf("Connected to server %s", server)

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		fmt.Println("Server response:", scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from server: %v", err)
	}
}