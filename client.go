package main

import (
	"fmt"
	"io"
	"net"
	"time"
)

// Client runs on the foreign server.
// It pre-connects tunnels to the Iran server (pool),
// and when activated, forwards traffic to the local service (xray/v2ray).

type Client struct {
	cfg *Config
}

func RunClient(cfg *Config) {
	c := &Client{cfg: cfg}
	c.start()
}

func (c *Client) start() {
	fmt.Printf("[CLIENT] Target Iran server: %s\n", c.cfg.RemoteAddr)
	fmt.Printf("[CLIENT] Local service: %s\n", c.cfg.LocalServiceAddr)
	fmt.Printf("[CLIENT] Pool size: %d\n", c.cfg.PoolSize)

	// Start heartbeat
	go c.heartbeatLoop()

	// Maintain pool
	c.maintainPool()
}

// maintainPool keeps PoolSize pre-connected tunnels alive at all times
func (c *Client) maintainPool() {
	sem := make(chan struct{}, c.cfg.PoolSize)

	for {
		// Fill up to PoolSize
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

// runTunnel opens one pre-connected tunnel to Iran server
func (c *Client) runTunnel() {
	// Connect to Iran server
	iranConn, err := c.dialIran()
	if err != nil {
		fmt.Printf("[CLIENT] Failed to connect to Iran: %v, retrying in %ds\n",
			err, c.cfg.ReconnectDelaySec)
		time.Sleep(time.Duration(c.cfg.ReconnectDelaySec) * time.Second)
		return
	}

	// Send auth + type = Tunnel
	header := append([]byte("T"), []byte(c.cfg.Token)...)
	_, err = iranConn.Write(header)
	if err != nil {
		iranConn.Close()
		return
	}

	// Wait for server to send "GO" (means a user connected)
	buf := make([]byte, 2)
	iranConn.SetReadDeadline(time.Now().Add(time.Duration(c.cfg.HeartbeatSec*10) * time.Second))
	_, err = io.ReadFull(iranConn, buf)
	if err != nil || string(buf) != "GO" {
		iranConn.Close()
		return
	}
	iranConn.SetReadDeadline(time.Time{})

	// User arrived! Connect to local service and bridge
	localConn, err := net.DialTimeout("tcp", c.cfg.LocalServiceAddr, 5*time.Second)
	if err != nil {
		fmt.Printf("[CLIENT] Failed to connect to local service: %v\n", err)
		iranConn.Close()
		return
	}

	fmt.Printf("[CLIENT] Tunnel active: %s ↔ local service\n", iranConn.RemoteAddr())
	bridge(iranConn, localConn)
	fmt.Printf("[CLIENT] Tunnel closed\n")
}

// dialIran connects to Iran server with retry
func (c *Client) dialIran() (net.Conn, error) {
	return net.DialTimeout("tcp", c.cfg.RemoteAddr, 10*time.Second)
}

// heartbeatLoop sends periodic pings to Iran server
func (c *Client) heartbeatLoop() {
	for {
		time.Sleep(time.Duration(c.cfg.HeartbeatSec) * time.Second)
		c.sendHeartbeat()
	}
}

func (c *Client) sendHeartbeat() {
	conn, err := c.dialIran()
	if err != nil {
		fmt.Printf("[CLIENT] Heartbeat failed: %v\n", err)
		return
	}
	defer conn.Close()

	header := append([]byte("H"), []byte(c.cfg.Token)...)
	conn.Write(header)

	buf := make([]byte, 2)
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	_, err = io.ReadFull(conn, buf)
	if err == nil && string(buf) == "OK" {
		fmt.Printf("[CLIENT] ♥ Heartbeat OK\n")
	}
}
