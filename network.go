package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/skip2/go-qrcode"
)

// getLocalIPs returns non-loopback active IPv4 addresses
func getLocalIPs() []string {
	var preferred []string
	var others []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return []string{"localhost"}
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		name := strings.ToLower(iface.Name)
		isVirtual := strings.Contains(name, "vethernet") ||
			strings.Contains(name, "wsl") ||
			strings.Contains(name, "docker") ||
			strings.Contains(name, "virtualbox") ||
			strings.Contains(name, "vmware")

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ipv4 := ip.To4()
			if ipv4 == nil {
				continue
			}

			ipStr := ipv4.String()
			if isVirtual {
				others = append(others, ipStr)
			} else {
				// Prioritize Wi-Fi and common local networks (192.168.x.x, 10.x.x.x, 172.x)
				if strings.HasPrefix(ipStr, "192.168.") || strings.HasPrefix(ipStr, "10.") || strings.HasPrefix(ipStr, "172.") {
					preferred = append(preferred, ipStr)
				} else {
					others = append(others, ipStr)
				}
			}
		}
	}

	all := append(preferred, others...)
	if len(all) == 0 {
		return []string{"localhost"}
	}
	return all
}

func getPrimaryPairingURL() string {
	ips := getLocalIPs()
	ip := "localhost"
	if len(ips) > 0 {
		ip = ips[0]
	}
	return fmt.Sprintf("http://%s:%d", ip, HTTP_PORT)
}

// handleAPIInfo returns system status and local network pairing details
func handleAPIInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ips := getLocalIPs()
	pairingURL := getPrimaryPairingURL()

	resp := map[string]interface{}{
		"ips":            ips,
		"http_port":      HTTP_PORT,
		"ws_port":        HTTP_PORT, // WebSockets served on primary HTTP port at /ws
		"legacy_ws_port": WS_PORT,
		"pairing_url":    pairingURL,
		"primary_ip":     ips[0],
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// handleAPIQR serves a dynamic PNG QR code of the pairing URL
func handleAPIQR(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		url = getPrimaryPairingURL()
	}

	pngBytes, err := qrcode.Encode(url, qrcode.Medium, 280)
	if err != nil {
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	_, _ = w.Write(pngBytes)
}
