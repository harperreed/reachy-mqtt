// ABOUTME: REST API client for the Reachy Mini's HTTP endpoints.
// ABOUTME: Provides on-demand queries and periodic state polling published to MQTT.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type RESTClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewRESTClient(apiURL string) *RESTClient {
	return &RESTClient{
		baseURL: apiURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Do executes an HTTP request and returns the raw response body.
func (r *RESTClient) Do(ctx context.Context, method, endpoint string, body []byte) ([]byte, int, error) {
	url := r.baseURL + endpoint

	var bodyReader io.Reader
	if body != nil {
		bodyReader = strings.NewReader(string(body))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	return data, resp.StatusCode, nil
}

// GetFullState fetches the complete robot state from the REST API.
func (r *RESTClient) GetFullState(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/state/full", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/state/full returned %d", status)
	}
	return data, nil
}

// GetDaemonStatus fetches the daemon status from the REST API.
func (r *RESTClient) GetDaemonStatus(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/daemon/status", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/daemon/status returned %d", status)
	}
	return data, nil
}

// GetMotorStatus fetches the motor status from the REST API.
func (r *RESTClient) GetMotorStatus(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/motors/status", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/motors/status returned %d", status)
	}
	return data, nil
}

// RunPoller periodically fetches state from the REST API and publishes to MQTT.
// Blocks until ctx is cancelled.
func (r *RESTClient) RunPoller(ctx context.Context, interval time.Duration, mqtt *MQTTBridge) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[rest] polling every %v", interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.pollAndPublish(ctx, mqtt)
		}
	}
}

func (r *RESTClient) pollAndPublish(ctx context.Context, mqttBridge *MQTTBridge) {
	// Poll full state
	if data, err := r.GetFullState(ctx); err == nil {
		mqttBridge.Publish("state", data)
	} else {
		log.Printf("[rest] poll state error: %v", err)
	}

	// Poll motor status
	if data, err := r.GetMotorStatus(ctx); err == nil {
		mqttBridge.Publish("motors", data)
	} else {
		log.Printf("[rest] poll motors error: %v", err)
	}
}

// HandleAPIRequest processes an MQTT API bridge request and returns the response.
func (r *RESTClient) HandleAPIRequest(ctx context.Context, reqData []byte) []byte {
	var req APIRequest
	if err := json.Unmarshal(reqData, &req); err != nil {
		resp := APIResponse{Error: fmt.Sprintf("invalid request: %v", err)}
		data, _ := json.Marshal(resp)
		return data
	}

	if req.Method == "" {
		req.Method = http.MethodGet
	}

	data, status, err := r.Do(ctx, req.Method, req.Endpoint, req.Body)

	var resp APIResponse
	resp.Endpoint = req.Endpoint
	resp.StatusCode = status

	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Data = data
	}

	result, _ := json.Marshal(resp)
	return result
}
