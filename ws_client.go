// ABOUTME: WebSocket client for the Reachy Mini robot's ws/sdk endpoint.
// ABOUTME: Maintains a persistent connection with auto-reconnect and exposes message channels.
package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type WSClient struct {
	url            string
	conn           *websocket.Conn
	mu             sync.Mutex
	incoming       chan []byte
	reconnectDelay time.Duration
}

func NewWSClient(url string, reconnectDelay time.Duration) *WSClient {
	return &WSClient{
		url:            url,
		incoming:       make(chan []byte, 256),
		reconnectDelay: reconnectDelay,
	}
}

// Run maintains the WebSocket connection, reconnecting on failure.
// Blocks until ctx is cancelled.
func (c *WSClient) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.close()
			return
		default:
		}

		err := c.connect(ctx)
		if err != nil {
			log.Printf("[ws] connect error: %v, retrying in %v", err, c.reconnectDelay)
			select {
			case <-ctx.Done():
				return
			case <-time.After(c.reconnectDelay):
				continue
			}
		}

		c.readLoop(ctx)

		log.Printf("[ws] disconnected, reconnecting in %v", c.reconnectDelay)
		select {
		case <-ctx.Done():
			c.close()
			return
		case <-time.After(c.reconnectDelay):
		}
	}
}

func (c *WSClient) connect(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, c.url, nil)
	if err != nil {
		return fmt.Errorf("dial %s: %w", c.url, err)
	}
	// Allow large messages (telemetry can be verbose)
	conn.SetReadLimit(1 << 20) // 1 MB

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	log.Printf("[ws] connected to %s", c.url)
	return nil
}

func (c *WSClient) readLoop(ctx context.Context) {
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("[ws] read error: %v", err)
			}
			return
		}

		select {
		case c.incoming <- data:
		default:
			// Drop message if channel is full (consumer too slow)
			log.Printf("[ws] dropping message, incoming channel full")
		}
	}
}

// Send writes a JSON message to the WebSocket connection.
func (c *WSClient) Send(ctx context.Context, data []byte) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}
	return conn.Write(ctx, websocket.MessageText, data)
}

// Messages returns a read-only channel of incoming raw JSON messages.
func (c *WSClient) Messages() <-chan []byte {
	return c.incoming
}

func (c *WSClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close(websocket.StatusNormalClosure, "shutdown")
		c.conn = nil
	}
}
