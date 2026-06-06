package main

import (
	"flag"
	"fmt"
	"os"
)

const version = "1.0.0"

func main() {
	mode := flag.String("mode", "", "Mode: server (Iran) or client (Foreign)")
	configPath := flag.String("config", "config.json", "Path to config file")
	showVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("GoTunnel v%s\n", version)
		os.Exit(0)
	}

	if *mode == "" {
		fmt.Println("GoTunnel - High Performance TCP Tunnel")
		fmt.Println("Usage:")
		fmt.Println("  gotunnel -mode server -config config.json   # Run on Iran server")
		fmt.Println("  gotunnel -mode client -config config.json   # Run on Foreign server")
		os.Exit(1)
	}

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to load config: %v\n", err)
		os.Exit(1)
	}

	switch *mode {
	case "server":
		fmt.Println("[GoTunnel] Starting in SERVER mode (Iran side)...")
		RunServer(cfg)
	case "client":
		fmt.Println("[GoTunnel] Starting in CLIENT mode (Foreign side)...")
		RunClient(cfg)
	default:
		fmt.Printf("[ERROR] Unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}
