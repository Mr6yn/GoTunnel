package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

type Client struct {
	cfg         *Config
	connected   int32 // atomic: 1=connected 0=disconnected
	lastPingMs  int64 // atomic: last ping in milliseconds
	totalServed int64 // atomic
	startTime   time.Time
}

var globalClient *Client

func RunClient(cfg *Config) {
	c := &Client{cfg: cfg, startTime: time.Now()}
	globalClient = c
	c.start()
}

func (c *Client) start() {
	fmt.Printf("[CLIENT] سرور ایران: %s\n", c.cfg.RemoteAddr)
	fmt.Printf("[CLIENT] سرویس محلی: %s\n", c.cfg.LocalServiceAddr)
	fmt.Printf("[CLIENT] Pool size: %d\n", c.cfg.PoolSize)

	go c.heartbeatLoop()
	go c.statusAPI()
	c.maintainPool()
}

func (c *Client) maintainPool() {
	sem := make(chan struct{}, c.cfg.PoolSize)
	for {
		for i := len(sem); i < c.cfg.PoolSize; i++ {
			sem <- struct{}{}
			go func() {
				defer func() { <-sem }()
				c.runTunnel()
			}()
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (c *Client) runTunnel() {
	iranConn, err := c.dialIran()
	if err != nil {
		atomic.StoreInt32(&c.connected, 0)
		time.Sleep(time.Duration(c.cfg.ReconnectDelaySec) * time.Second)
		return
	}

	header := append([]byte("T"), []byte(c.cfg.Token)...)
	_, err = iranConn.Write(header)
	if err != nil {
		iranConn.Close()
		return
	}

	buf := make([]byte, 2)
	iranConn.SetReadDeadline(time.Now().Add(time.Duration(c.cfg.HeartbeatSec*10) * time.Second))
	_, err = io.ReadFull(iranConn, buf)
	if err != nil || string(buf) != "GO" {
		iranConn.Close()
		return
	}
	iranConn.SetReadDeadline(time.Time{})

	localConn, err := net.DialTimeout("tcp", c.cfg.LocalServiceAddr, 5*time.Second)
	if err != nil {
		fmt.Printf("[CLIENT] سرویس محلی در دسترس نیست: %v\n", err)
		iranConn.Close()
		return
	}

	atomic.AddInt64(&c.totalServed, 1)
	bridge(iranConn, localConn)
}

func (c *Client) dialIran() (net.Conn, error) {
	return net.DialTimeout("tcp", c.cfg.RemoteAddr, 10*time.Second)
}

func (c *Client) heartbeatLoop() {
	for {
		c.sendHeartbeat()
		time.Sleep(time.Duration(c.cfg.HeartbeatSec) * time.Second)
	}
}

func (c *Client) sendHeartbeat() {
	start := time.Now()
	conn, err := c.dialIran()
	if err != nil {
		atomic.StoreInt32(&c.connected, 0)
		fmt.Printf("[CLIENT] ✗ Heartbeat قطع: %v\n", err)
		return
	}
	defer conn.Close()

	header := append([]byte("H"), []byte(c.cfg.Token)...)
	conn.Write(header)

	buf := make([]byte, 2)
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	_, err = io.ReadFull(conn, buf)
	if err == nil && string(buf) == "OK" {
		pingMs := time.Since(start).Milliseconds()
		atomic.StoreInt32(&c.connected, 1)
		atomic.StoreInt64(&c.lastPingMs, pingMs)
		fmt.Printf("[CLIENT] ♥ Heartbeat OK — ping: %dms\n", pingMs)
	} else {
		atomic.StoreInt32(&c.connected, 0)
	}
}

// statusAPI exposes HTTP status on StatusPort
func (c *Client) statusAPI() {
	if c.cfg.StatusPort == 0 {
		return
	}
	addr := fmt.Sprintf("127.0.0.1:%d", c.cfg.StatusPort)
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"mode":         "client",
			"connected":    atomic.LoadInt32(&c.connected) == 1,
			"ping_ms":      atomic.LoadInt64(&c.lastPingMs),
			"total_served": atomic.LoadInt64(&c.totalServed),
			"remote":       c.cfg.RemoteAddr,
			"uptime":       time.Since(c.startTime).Round(time.Second).String(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	})
	http.ListenAndServe(addr, nil)
}
