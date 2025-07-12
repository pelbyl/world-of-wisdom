package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"world-of-wisdom/pkg/pow"
	"world-of-wisdom/pkg/wisdom"
)

type Server struct {
	listener       net.Listener
	quoteProvider  *wisdom.QuoteProvider
	difficulty     int
	timeout        time.Duration
	mu             sync.RWMutex
	activeConns    sync.WaitGroup
	shutdownChan   chan struct{}
}

type Config struct {
	Port       string
	Difficulty int
	Timeout    time.Duration
}

func NewServer(cfg Config) (*Server, error) {
	listener, err := net.Listen("tcp", cfg.Port)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", cfg.Port, err)
	}

	return &Server{
		listener:      listener,
		quoteProvider: wisdom.NewQuoteProvider(),
		difficulty:    cfg.Difficulty,
		timeout:       cfg.Timeout,
		shutdownChan:  make(chan struct{}),
	}, nil
}

func (s *Server) Start() error {
	log.Printf("Server listening on %s with difficulty %d", s.listener.Addr(), s.difficulty)

	for {
		select {
		case <-s.shutdownChan:
			return nil
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.shutdownChan:
					return nil
				default:
					log.Printf("Failed to accept connection: %v", err)
					continue
				}
			}

			s.activeConns.Add(1)
			go s.handleConnection(conn)
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.activeConns.Done()
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()
	log.Printf("New connection from %s", clientAddr)

	conn.SetDeadline(time.Now().Add(s.timeout))

	challenge, err := pow.GenerateChallenge(s.getDifficulty())
	if err != nil {
		log.Printf("Failed to generate challenge: %v", err)
		conn.Write([]byte("Error: Failed to generate challenge\n"))
		return
	}

	_, err = conn.Write([]byte(challenge.String() + "\n"))
	if err != nil {
		log.Printf("Failed to send challenge to %s: %v", clientAddr, err)
		return
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		log.Printf("Client %s disconnected or timed out", clientAddr)
		return
	}

	response := strings.TrimSpace(scanner.Text())

	if pow.VerifyPoW(challenge.Seed, response, challenge.Difficulty) {
		log.Printf("Client %s solved the challenge", clientAddr)
		quote := s.quoteProvider.GetRandomQuote()
		conn.Write([]byte(quote + "\n"))
	} else {
		log.Printf("Client %s failed the challenge", clientAddr)
		conn.Write([]byte("Error: Invalid proof of work\n"))
	}
}

func (s *Server) getDifficulty() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.difficulty
}

func (s *Server) SetDifficulty(difficulty int) error {
	if difficulty < 1 || difficulty > 6 {
		return fmt.Errorf("difficulty must be between 1 and 6")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.difficulty = difficulty
	log.Printf("Difficulty updated to %d", difficulty)
	return nil
}

func (s *Server) Shutdown() error {
	log.Println("Shutting down server...")
	close(s.shutdownChan)
	
	err := s.listener.Close()
	if err != nil {
		return fmt.Errorf("failed to close listener: %w", err)
	}

	done := make(chan struct{})
	go func() {
		s.activeConns.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All connections closed")
	case <-time.After(10 * time.Second):
		log.Println("Timeout waiting for connections to close")
	}

	return nil
}