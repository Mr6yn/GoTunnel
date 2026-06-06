package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	// Common
	Token      string `json:"token"`
	StatusPort int    `json:"status_port"` // local HTTP status API port (e.g. 9999)

	// Server side (Iran)
	ServerBindAddr string `json:"server_bind_addr"`
	ServerPorts    []int  `json:"server_ports"`

	// Client side (Foreign)
	RemoteAddr       string `json:"remote_addr"`
	LocalServiceAddr string `json:"local_service_addr"`

	// Tunnel settings
	PoolSize          int `json:"pool_size"`
	HeartbeatSec      int `json:"heartbeat_sec"`
	ReconnectDelaySec int `json:"reconnect_delay_sec"`
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
		StatusPort:        9999,
	}

	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
