package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	// Common
	Token string `json:"token"` // Secret token for auth

	// Server side (Iran)
	ServerBindAddr  string `json:"server_bind_addr"`  // e.g. "0.0.0.0:8443"
	ServerPorts     []int  `json:"server_ports"`      // Ports to expose to users e.g. [443, 80, 8080]

	// Client side (Foreign)
	RemoteAddr      string `json:"remote_addr"`       // Iran server IP:port e.g. "1.2.3.4:8443"
	LocalServiceAddr string `json:"local_service_addr"` // Local xray/v2ray e.g. "127.0.0.1:10000"

	// Tunnel settings
	PoolSize        int    `json:"pool_size"`         // Pre-connected tunnel pool size (default: 8)
	HeartbeatSec    int    `json:"heartbeat_sec"`     // Heartbeat interval in seconds (default: 10)
	ReconnectDelaySec int  `json:"reconnect_delay_sec"` // Reconnect delay (default: 3)
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := &Config{
		PoolSize:          8,
		HeartbeatSec:      10,
		ReconnectDelaySec: 3,
	}

	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
