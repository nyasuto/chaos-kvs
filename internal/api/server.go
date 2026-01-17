package api

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"chaos-kvs/internal/cluster"
	"chaos-kvs/internal/logger"
	"chaos-kvs/internal/scenario"

	"golang.org/x/net/websocket"
)

//go:embed static/*
var staticFiles embed.FS

// Server はAPIサーバー
type Server struct {
	addr    string
	cluster *cluster.Cluster
	engine  *scenario.Engine
	config  scenario.Config

	mu        sync.RWMutex
	running   bool
	wsClients map[*websocket.Conn]bool

	server *http.Server
}

// NewServer は新しいAPIサーバーを作成する
func NewServer(addr string) *Server {
	return &Server{
		addr:      addr,
		wsClients: make(map[*websocket.Conn]bool),
	}
}

// Start はサーバーを開始する
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/nodes", s.handleNodes)
	mux.HandleFunc("/api/metrics", s.handleMetrics)
	mux.HandleFunc("/api/scenario/start", s.handleScenarioStart)
	mux.HandleFunc("/api/scenario/stop", s.handleScenarioStop)
	mux.HandleFunc("/api/presets", s.handlePresets)

	// WebSocket
	mux.Handle("/ws", websocket.Handler(s.handleWebSocket))

	// Static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("failed to get static files: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	// バックグラウンドでメトリクス配信
	go s.broadcastLoop(ctx)

	logger.Info("", "API Server starting on http://%s", s.addr)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
	}()

	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// StatusResponse はステータスレスポンス
type StatusResponse struct {
	Running        bool   `json:"running"`
	ScenarioName   string `json:"scenario_name,omitempty"`
	NodeCount      int    `json:"node_count"`
	RunningNodes   int    `json:"running_nodes"`
	StoppedNodes   int    `json:"stopped_nodes"`
	SuspendedNodes int    `json:"suspended_nodes"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	resp := StatusResponse{
		Running: s.running,
	}

	if s.config.Name != "" {
		resp.ScenarioName = s.config.Name
	}

	if s.cluster != nil {
		resp.NodeCount = s.cluster.Size()
		resp.RunningNodes = s.cluster.RunningCount()
		for _, n := range s.cluster.Nodes() {
			switch n.Status().String() {
			case "Stopped":
				resp.StoppedNodes++
			case "Suspended":
				resp.SuspendedNodes++
			}
		}
	}

	s.writeJSON(w, resp)
}

// NodeInfo はノード情報
type NodeInfo struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Size   int    `json:"size"`
	Delay  string `json:"delay,omitempty"`
}

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var nodes []NodeInfo
	if s.cluster != nil {
		for _, n := range s.cluster.Nodes() {
			info := NodeInfo{
				ID:     n.ID(),
				Status: n.Status().String(),
				Size:   n.Size(),
			}
			if d := n.Delay(); d > 0 {
				info.Delay = d.String()
			}
			nodes = append(nodes, info)
		}
	}

	s.writeJSON(w, nodes)
}

// MetricsResponse はメトリクスレスポンス
type MetricsResponse struct {
	TotalRequests   uint64  `json:"total_requests"`
	SuccessRequests uint64  `json:"success_requests"`
	FailedRequests  uint64  `json:"failed_requests"`
	RPS             float64 `json:"rps"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
	P99LatencyMs    float64 `json:"p99_latency_ms"`
	ErrorRate       float64 `json:"error_rate"`
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	engine := s.engine
	s.mu.RUnlock()

	resp := MetricsResponse{}

	// Note: Engine doesn't expose metrics directly yet
	// This would need to be enhanced in the scenario package
	_ = engine // suppress unused variable warning

	s.writeJSON(w, resp)
}

// ScenarioRequest はシナリオ開始リクエスト
type ScenarioRequest struct {
	Preset   string `json:"preset"`
	Duration string `json:"duration,omitempty"`
	Nodes    int    `json:"nodes,omitempty"`
}

func (s *Server) handleScenarioStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		http.Error(w, "Scenario already running", http.StatusConflict)
		return
	}

	// プリセット取得
	config, ok := scenario.GetPreset(req.Preset)
	if !ok {
		config = scenario.QuickScenario()
	}

	// オーバーライド
	if req.Duration != "" {
		if d, err := time.ParseDuration(req.Duration); err == nil {
			config.Duration = d
		}
	}
	if req.Nodes > 0 {
		config.NodeCount = req.Nodes
	}

	s.config = config
	s.cluster = cluster.New()
	s.engine = scenario.New(config)
	s.running = true
	s.mu.Unlock()

	// バックグラウンドで実行
	go func() {
		ctx := context.Background()
		result, err := s.engine.Run(ctx)

		s.mu.Lock()
		s.running = false
		s.mu.Unlock()

		if err != nil {
			logger.Error("", "Scenario failed: %v", err)
		} else {
			logger.Info("", "Scenario completed: %d requests", result.TotalRequests)
		}

		s.broadcast(map[string]interface{}{
			"type":   "scenario_complete",
			"result": result,
		})
	}()

	s.writeJSON(w, map[string]string{"status": "started", "scenario": config.Name})
}

func (s *Server) handleScenarioStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		http.Error(w, "No scenario running", http.StatusBadRequest)
		return
	}
	// Note: Would need to add cancellation support to scenario.Engine
	s.mu.Unlock()

	s.writeJSON(w, map[string]string{"status": "stop requested"})
}

// PresetInfo はプリセット情報
type PresetInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (s *Server) handlePresets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	presets := []PresetInfo{
		{"basic", "カオスなしの基本負荷テスト"},
		{"resilience", "ノードkillと復旧のテスト"},
		{"latency", "レイテンシ注入テスト"},
		{"stress", "高負荷ストレステスト"},
		{"quick", "短時間の動作確認"},
	}

	s.writeJSON(w, presets)
}

// WebSocket handling
func (s *Server) handleWebSocket(ws *websocket.Conn) {
	s.mu.Lock()
	s.wsClients[ws] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.wsClients, ws)
		s.mu.Unlock()
		_ = ws.Close()
	}()

	// Keep connection alive
	for {
		var msg string
		if err := websocket.Message.Receive(ws, &msg); err != nil {
			break
		}
	}
}

func (s *Server) broadcast(data interface{}) {
	s.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(s.wsClients))
	for ws := range s.wsClients {
		clients = append(clients, ws)
	}
	s.mu.RUnlock()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	for _, ws := range clients {
		_ = websocket.Message.Send(ws, string(jsonData))
	}
}

func (s *Server) broadcastLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.RLock()
			if !s.running {
				s.mu.RUnlock()
				continue
			}

			status := StatusResponse{
				Running: s.running,
			}
			if s.config.Name != "" {
				status.ScenarioName = s.config.Name
			}
			if s.cluster != nil {
				status.NodeCount = s.cluster.Size()
				status.RunningNodes = s.cluster.RunningCount()
			}
			s.mu.RUnlock()

			s.broadcast(map[string]interface{}{
				"type":   "status",
				"status": status,
			})
		}
	}
}

func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("", "Failed to encode JSON: %v", err)
	}
}
