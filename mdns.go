package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/grandcat/zeroconf"
)

// TODO: share clipboard between PCs that in same LAN
func StartMDNSServer(ctx context.Context, port int) error {
	hostname, err := os.Hostname()
	if err != nil {
		log.Println("[Warn] failed to get hostname:", err)
		hostname = "evercliping"
	}

	server, err := zeroconf.Register(hostname, "_evercliping._tcp", "local.", port, []string{"desc=EverCliping Server", "version=0.1"}, nil)
	if err != nil {
		return fmt.Errorf("failed to register mdns server: %w", err)
	}

	log.Println("Starting mDNS server...")
	<-ctx.Done()
	server.Shutdown()
	log.Println("Shutdown mDNS server.")
	return nil
}
