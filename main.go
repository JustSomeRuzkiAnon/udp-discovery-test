package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

const probePrefix = "UDPDISC1 "
const probeIDLength = 32

func configuredPort(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		log.Fatalf("%s must be a port number from 1 to 65535", name)
	}
	return port
}

func receiveProbes(conn *net.UDPConn, ready *atomic.Bool) {
	defer ready.Store(false)
	defer conn.Close()
	packet := make([]byte, 65535)
	for {
		size, source, err := conn.ReadFromUDPAddrPort(packet)
		if err != nil {
			log.Printf("udp_read_error error=%q", err)
			return
		}
		if size != len(probePrefix)+probeIDLength || !bytes.HasPrefix(packet[:size], []byte(probePrefix)) {
			continue
		}
		id := packet[len(probePrefix):size]
		var decoded [probeIDLength / 2]byte
		if _, err := hex.Decode(decoded[:], id); err == nil {
			log.Printf("udp_probe id=%s source_ip=%s source_port=%d", id, source.Addr(), source.Port())
		}
	}
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)
	httpPort := configuredPort("PORT", 8080)
	udpPort := configuredPort("UDP_PORT", httpPort)
	var udpReady atomic.Bool
	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: udpPort})
	if err != nil {
		log.Printf("udp_bind_error port=%d error=%q", udpPort, err)
	} else {
		log.Printf("udp_listening address=%s", udp.LocalAddr())
		udpReady.Store(true)
		go receiveProbes(udp, &udpReady)
	}

	health := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"http_ok":       true,
			"udp_listening": udpReady.Load(),
			"udp_port":      udpPort,
		})
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", health)
	mux.HandleFunc("GET /healthz", health)
	server := &http.Server{
		Addr:              "0.0.0.0:" + strconv.Itoa(httpPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("http_listening address=%s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
