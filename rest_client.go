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

	// Poll Direction of Arrival
	if data, err := r.GetDoA(ctx); err == nil {
		mqttBridge.Publish("doa", data)
	} else {
		log.Printf("[rest] poll doa error: %v", err)
	}

	// Poll current app status
	if data, err := r.GetCurrentAppStatus(ctx); err == nil {
		mqttBridge.Publish("app_status", data)
	} else {
		log.Printf("[rest] poll app status error: %v", err)
	}
}

// GetDoA fetches the Direction of Arrival from the microphone array.
func (r *RESTClient) GetDoA(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/state/doa", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/state/doa returned %d", status)
	}
	return data, nil
}

// GetCameraSpecs fetches camera intrinsics for CV pipelines.
func (r *RESTClient) GetCameraSpecs(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/camera/specs", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/camera/specs returned %d", status)
	}
	return data, nil
}

// PlayEmotion triggers a pre-built emotion animation by name.
func (r *RESTClient) PlayEmotion(ctx context.Context, name string) ([]byte, error) {
	endpoint := fmt.Sprintf("/api/move/play/recorded-move-dataset/pollen-robotics/reachy-mini-emotions-library/%s", name)
	data, status, err := r.Do(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST %s returned %d: %s", endpoint, status, string(data))
	}
	return data, nil
}

// PlayDance triggers a pre-built dance animation by name.
func (r *RESTClient) PlayDance(ctx context.Context, name string) ([]byte, error) {
	endpoint := fmt.Sprintf("/api/move/play/recorded-move-dataset/pollen-robotics/reachy-mini-dances-library/%s", name)
	data, status, err := r.Do(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST %s returned %d: %s", endpoint, status, string(data))
	}
	return data, nil
}

// ListEmotions lists available emotion animations.
func (r *RESTClient) ListEmotions(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/move/recorded-move-datasets/list/pollen-robotics/reachy-mini-emotions-library", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("list emotions returned %d", status)
	}
	return data, nil
}

// ListDances lists available dance animations.
func (r *RESTClient) ListDances(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/move/recorded-move-datasets/list/pollen-robotics/reachy-mini-dances-library", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("list dances returned %d", status)
	}
	return data, nil
}

// GotoMove sends a smooth interpolated movement via REST.
func (r *RESTClient) GotoMove(ctx context.Context, body []byte) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodPost, "/api/move/goto", body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/move/goto returned %d: %s", status, string(data))
	}
	return data, nil
}

// StopMove stops a running move by UUID.
func (r *RESTClient) StopMove(ctx context.Context, uuid string) ([]byte, error) {
	body, _ := json.Marshal(map[string]string{"uuid": uuid})
	data, status, err := r.Do(ctx, http.MethodPost, "/api/move/stop", body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/move/stop returned %d", status)
	}
	return data, nil
}

// ListRunningMoves lists currently running move tasks.
func (r *RESTClient) ListRunningMoves(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/move/running", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/move/running returned %d", status)
	}
	return data, nil
}

// DaemonStart starts the robot daemon.
func (r *RESTClient) DaemonStart(ctx context.Context, wakeUp bool) ([]byte, error) {
	body, _ := json.Marshal(map[string]bool{"wake_up": wakeUp})
	data, status, err := r.Do(ctx, http.MethodPost, "/api/daemon/start", body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/daemon/start returned %d", status)
	}
	return data, nil
}

// DaemonStop stops the robot daemon.
func (r *RESTClient) DaemonStop(ctx context.Context, gotoSleep bool) ([]byte, error) {
	body, _ := json.Marshal(map[string]bool{"goto_sleep": gotoSleep})
	data, status, err := r.Do(ctx, http.MethodPost, "/api/daemon/stop", body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/daemon/stop returned %d", status)
	}
	return data, nil
}

// DaemonRestart restarts the robot daemon.
func (r *RESTClient) DaemonRestart(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodPost, "/api/daemon/restart", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/daemon/restart returned %d", status)
	}
	return data, nil
}

// GetCurrentAppStatus gets the status of the currently running app.
func (r *RESTClient) GetCurrentAppStatus(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/apps/current-app-status", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/apps/current-app-status returned %d", status)
	}
	return data, nil
}

// ListApps lists all available apps.
func (r *RESTClient) ListApps(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/apps/list-available", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/apps/list-available returned %d", status)
	}
	return data, nil
}

// StartApp starts an app by name.
func (r *RESTClient) StartApp(ctx context.Context, name string) ([]byte, error) {
	endpoint := fmt.Sprintf("/api/apps/start-app/%s", name)
	data, status, err := r.Do(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST %s returned %d", endpoint, status)
	}
	return data, nil
}

// StopApp stops the currently running app.
func (r *RESTClient) StopApp(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodPost, "/api/apps/stop-current-app", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/apps/stop-current-app returned %d", status)
	}
	return data, nil
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
