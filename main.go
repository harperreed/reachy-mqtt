// ABOUTME: Entry point for the reachy-mqtt bridge daemon.
// ABOUTME: Wires together WebSocket, MQTT, and REST clients with signal-based graceful shutdown.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configPath := flag.String("config", "", "path to JSON config file")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	log.Printf("reachy-mqtt bridge starting")
	log.Printf("  robot: %s at %s", cfg.ReachyName, cfg.WSURL())
	log.Printf("  mqtt:  %s (client: %s)", cfg.MQTTBroker, cfg.MQTTClientID)
	log.Printf("  rest:  polling %s every %v", cfg.APIURL(), cfg.RESTPollDuration())
	log.Printf("  throttle: max %v, heartbeat %v", cfg.MaxPublishInterval(), cfg.HeartbeatDuration())

	// Create context with signal-based cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Initialize components
	ws := NewWSClient(cfg.WSURL(), cfg.WSReconnectDuration())
	rest := NewRESTClient(cfg.APIURL())

	mqttBridge, err := NewMQTTBridge(cfg)
	if err != nil {
		log.Fatalf("mqtt init: %v", err)
	}

	router := NewRouter(ws, mqttBridge, rest, cfg.MaxPublishInterval(), cfg.HeartbeatDuration())

	// Start WebSocket client (auto-reconnecting)
	go ws.Run(ctx)

	// Start REST poller
	go rest.RunPoller(ctx, cfg.RESTPollDuration(), mqttBridge)

	// Start router (main message loop)
	go router.Run(ctx)

	log.Printf("reachy-mqtt bridge running (ctrl+c to stop)")

	// Wait for shutdown signal
	sig := <-sigCh
	log.Printf("received %v, shutting down", sig)

	cancel()
	mqttBridge.Close()

	log.Printf("reachy-mqtt bridge stopped")
}
