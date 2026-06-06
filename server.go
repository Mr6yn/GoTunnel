package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Server runs on the Iran server.

type Server struct {
	cfg          *Config
	clientPool   chan net.Conn
	mu           sync.Mutex
	poolReady    int64 // atomic count of ready tunnels
	totalServed  int64 // atomic count of served connections
	lastClientIP string
	lastSeen     time.Time
	connected    bool
	startTime    time.Time
}

var globalServer *Server

func RunServer(cfg *Config) {
	s := &Server{
		cfg:       cfg,
		clientPool: make(chan net.Conn, cfg.PoolSize*2),
		startTime: time.Now(),
	}
	globalServer = s
	s.start()
}

func (s *Server) start() {
	go s.listenControl()
	for _, port := range s.cfg.ServerPorts {
		go s.listenUserPort(port)
	}
	go s.statusAPI()
	go s.healthReporter()
	select {}
}

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
			time.Sleep(time.Second)
			continue
		}
		go s.handleIncoming(conn)
	}
}

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

	conn.SetDeadline(time.Time{})

	switch msgType {
	case 'T': // Tunnel pre-connect
		select {
		case s.clientPool <- conn:
			atomic.AddInt64(&s.poolReady, 1)
			s.mu.Lock()
			s.lastClientIP = conn.RemoteAddr().String()
			s.lastSeen = time.Now()
			s.connected = true
			s.mu.Unlock()
		default:
			conn.Close()
		}
	case 'H': // Heartbeat
		s.mu.Lock()
		s.lastClientIP = conn.RemoteAddr().String()
		s.lastSeen = time.Now()
		s.connected = true
		s.mu.Unlock()
		conn.Write([]byte("OK"))
		conn.Close()
	case 'S': // Status ping from client
		s.mu.Lock()
		pool := len(s.clientPool)
		served := atomic.LoadInt64(&s.totalServed)
		uptime := time.Since(s.startTime).Round(time.Second).String()
		s.mu.Unlock()
		resp := fmt.Sprintf(`{"pool":%d,"served":%d,"uptime":"%s"}`, pool, served, uptime)
		conn.Write([]byte(resp))
		conn.Close()
	default:
		conn.Close()
	}
}

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

func (s *Server) bridgeUser(userConn net.Conn) {
	defer userConn.Close()

	var tunnelConn net.Conn
	select {
	case tunnelConn = <-s.clientPool:
		atomic.AddInt64(&s.poolReady, -1)
	case <-time.After(5 * time.Second):
		fmt.Println("[SERVER] No tunnel available, dropping user connection")
		return
	}

	atomic.AddInt64(&s.totalServed, 1)
	tunnelConn.Write([]byte("GO"))
	bridge(userConn, tunnelConn)
}

// statusAPI exposes HTTP status endpoint on StatusPort
func (s *Server) statusAPI() {
	if s.cfg.StatusPort == 0 {
		return
	}
	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.StatusPort)
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		connected := s.connected
		lastSeen := s.lastSeen
		clientIP := s.lastClientIP
		s.mu.Unlock()

		// mark disconnected if no heartbeat for 3x heartbeat interval
		if time.Since(lastSeen) > time.Duration(s.cfg.HeartbeatSec*3)*time.Second {
			connected = false
		}

		data := map[string]interface{}{
			"mode":        "server",
			"connected":   connected,
			"pool_ready":  atomic.LoadInt64(&s.poolReady),
			"total_served": atomic.LoadInt64(&s.totalServed),
			"client_ip":   clientIP,
			"last_seen":   lastSeen.Format("15:04:05"),
			"uptime":      time.Since(s.startTime).Round(time.Second).String(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	})
	http.ListenAndServe(addr, nil)
}

func (s *Server) healthReporter() {
	for {
		time.Sleep(30 * time.Second)
		s.mu.Lock()
		connected := s.connected
		lastSeen := s.lastSeen
		s.mu.Unlock()
		if time.Since(lastSeen) > time.Duration(s.cfg.HeartbeatSec*3)*time.Second {
			connected = false
		}
		status := "✓ وصل"
		if !connected {
			status = "✗ قطع"
		}
		fmt.Printf("[SERVER] وضعیت: %s | Pool: %d | کل اتصال: %d\n",
			status, len(s.clientPool), atomic.LoadInt64(&s.totalServed))
	}
}

func bridge(a, b net.Conn) {
	defer a.Close()
	defer b.Close()

	done := make(chan struct{}, 2)
	go func() { io.Copy(a, b); done <- struct{}{} }()
	go func() { io.Copy(b, a); done <- struct{}{} }()
	<-done
}
