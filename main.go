package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

func main() {
	mode       := flag.String("mode",    "",            "server | client | status")
	configPath := flag.String("config",  "/opt/gotunnel/config.json", "config file path")
	flag.Parse()

	switch *mode {
	case "server":
		cfg := mustLoadConfig(*configPath)
		fmt.Println("[GoTunnel] در حال اجرا — سرور ایران...")
		RunServer(cfg)

	case "client":
		cfg := mustLoadConfig(*configPath)
		fmt.Println("[GoTunnel] در حال اجرا — سرور خارج...")
		RunClient(cfg)

	case "status":
		showStatus(*configPath)

	default:
		fmt.Printf("GoTunnel v%s\n", version)
		fmt.Println("دستورات:")
		fmt.Println("  gotunnel -mode server   ← سرور ایران")
		fmt.Println("  gotunnel -mode client   ← سرور خارج")
		fmt.Println("  gotunnel -mode status   ← وضعیت تانل")
		os.Exit(1)
	}
}

func mustLoadConfig(path string) *Config {
	cfg, err := LoadConfig(path)
	if err != nil {
		fmt.Printf("[ERROR] خواندن کانفیگ ناموفق: %v\n", err)
		os.Exit(1)
	}
	return cfg
}

// showStatus calls the local status API and prints a pretty report
func showStatus(configPath string) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		fmt.Println("❌ کانفیگ پیدا نشد")
		os.Exit(1)
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/status", cfg.StatusPort)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Println("❌ سرویس GoTunnel در حال اجرا نیست")
		fmt.Println("   دستور: systemctl start gotunnel")
		os.Exit(1)
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)

	connected := data["connected"].(bool)
	uptime     := data["uptime"].(string)
	mode       := data["mode"].(string)

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║         GoTunnel — وضعیت تانل        ║")
	fmt.Println("╠══════════════════════════════════════╣")

	if connected {
		fmt.Println("║  اتصال:  ✅ وصل                      ║")
	} else {
		fmt.Println("║  اتصال:  ❌ قطع                      ║")
	}

	fmt.Printf("║  نوع:    %-30s║\n", modeLabel(mode))
	fmt.Printf("║  آپتایم: %-30s║\n", uptime)

	if mode == "client" {
		pingMs  := int64(data["ping_ms"].(float64))
		served  := int64(data["total_served"].(float64))
		remote  := data["remote"].(string)
		fmt.Printf("║  پینگ:   %-30s║\n", fmt.Sprintf("%dms", pingMs))
		fmt.Printf("║  ایران:  %-30s║\n", remote)
		fmt.Printf("║  اتصال‌ها: %-29s║\n", fmt.Sprintf("%d اتصال کاربر", served))
	} else {
		pool   := int64(data["pool_ready"].(float64))
		served := int64(data["total_served"].(float64))
		ip     := data["client_ip"].(string)
		lastSeen := data["last_seen"].(string)
		fmt.Printf("║  Pool:   %-30s║\n", fmt.Sprintf("%d تانل آماده", pool))
		fmt.Printf("║  سرور خارج: %-27s║\n", ip)
		fmt.Printf("║  آخرین ارتباط: %-24s║\n", lastSeen)
		fmt.Printf("║  اتصال‌ها: %-29s║\n", fmt.Sprintf("%d اتصال کاربر", served))
	}

	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()

	if !connected {
		os.Exit(1)
	}
}

func modeLabel(mode string) string {
	if mode == "server" {
		return "سرور ایران"
	}
	return "سرور خارج"
}
