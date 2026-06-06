package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Server runs on the Iran server.
// It listens for:
//   1. Control connections from the foreign client
//   2. User connections on exposed ports
// Then bridges them together.

type Server struct {
	cfg        *Config
	clientPool chan net.Conn // pre-connected tunnels from foreign client
	mu         sync.Mutex
}

func RunServer(cfg *Config) {
	s := &Server{
		cfg:        cfg,
		clientPool: make(chan net.Conn, cfg.PoolSize*2),
	}
	s.start()
}

func (s *Server) start() {
	// Start control listener (foreign client connects here)
	go s.listenControl()

	// Start user-facing port listeners
	for _, port := range s.cfg.ServerPorts {
		go s.listenUserPort(port)
	}

	// Start health reporter
	go s.healthReporter()

	// Block forever
	select {}
}

// listenControl waits for the foreign client to connect and register tunnels
func (s *Server) listenControl() {
	ln, err := net.Listen("tcp", s.cfg.ServerBindAddr)
	if err != nil {
		fmt.Printf("[SERVER] Failed to bind control port %s: %v\n", s.cfg.ServerBindAddr, err)
		return
	}
	fmt.Printf("[SERVER] Control listener on %s\n", s.cfg.ServerBindAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("[SERVER] Accept error: %v\n", err)
			time.Sleep(time.Second)
			continue
		}
		go s.handleIncoming(conn)
	}
}

// handleIncoming reads the first byte to determine connection type:
// 'T' = tunnel connection (pre-pooled, waiting for user)
// 'H' = heartbeat ping
func (s *Server) handleIncoming(conn net.Conn) {
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	buf := make([]byte, 1+len(s.cfg.Token))
	_, err := io.ReadFull(conn, buf)
	if err != nil {
		conn.Close()
		return
	}

	msgType := buf[0]
	token := string(buf[1:])

	if token != s.cfg.Token {
		fmt.Printf("[SERVER] Auth failed from %s\n", conn.RemoteAddr())
		conn.Close()
		return
	}

	conn.SetDeadline(time.Time{}) // clear deadline

	switch msgType {
	case 'T': // Tunnel - add to pool
		select {
		case s.clientPool <- conn:
			// ok
		default:
			// Pool full, close this one
			conn.Close()
		}
	case 'H': // Heartbeat
		conn.Write([]byte("OK"))
		conn.Close()
	default:
		conn.Close()
	}
}

// listenUserPort listens for end-user connections on a given port
func (s *Server) listenUserPort(port int) {
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("[SERVER] Failed to listen on port %d: %v\n", port, err)
		return
	}
	fmt.Printf("[SERVER] Listening for users on port %d\n", port)

	for {
		userConn, err := ln.Accept()
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		go s.bridgeUser(userConn)
	}
}

// bridgeUser grabs a pre-connected tunnel and bridges user ↔ foreign server
func (s *Server) bridgeUser(userConn net.Conn) {
	defer userConn.Close()

	// Wait up to 5s for an available tunnel
	var tunnelConn net.Conn
	select {
	case tunnelConn = <-s.clientPool:
	case <-time.After(5 * time.Second):
		fmt.Println("[SERVER] No tunnel available, dropping user connection")
		return
	}

	// Signal client that this tunnel is now active
	tunnelConn.Write([]byte("GO"))

	// Bridge the two connections
	bridge(userConn, tunnelConn)
}

func (s *Server) healthReporter() {
	for {
		time.Sleep(30 * time.Second)
		fmt.Printf("[SERVER] Pool size: %d tunnels ready\n", len(s.clientPool))
	}
}

// bridge copies data between two connections bidirectionally
func bridge(a, b net.Conn) {
	defer a.Close()
	defer b.Close()

	done := make(chan struct{}, 2)

	go func() {
		io.Copy(a, b)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(b, a)
		done <- struct{}{}
	}()

	<-done
}
