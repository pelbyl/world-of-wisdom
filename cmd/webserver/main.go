package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"world-of-wisdom/internal/webserver"
)

func main() {
	var (
		port = flag.String("port", ":8081", "Web server port")
		tcpServer = flag.String("tcp-server", "localhost:8080", "TCP server address")
	)
	flag.Parse()

	server := webserver.NewWebServer(*tcpServer)

	http.HandleFunc("/ws", server.HandleWebSocket)
	http.HandleFunc("/api/stats", server.HandleStats)
	http.HandleFunc("/api/simulate", server.HandleSimulateClient)

	go server.Start()

	log.Printf("Web server listening on %s", *port)
	go func() {
		if err := http.ListenAndServe(*port, nil); err != nil {
			log.Fatalf("Web server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down web server...")
	server.Stop()
}