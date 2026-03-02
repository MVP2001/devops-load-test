package models

import (
	"sync"
	"time"
)

type TargetVM struct {
	ID       string `json:"id" binding:"required"`
	IP       string `json:"ip" binding:"required"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type LoadConfig struct {
	VM              TargetVM               `json:"vm" binding:"required"`
	Duration        time.Duration          `json:"duration" binding:"required"`
	Intensity       string                 `json:"intensity"`
	ConcurrentUsers int                    `json:"concurrent_users"`
	CustomParams    map[string]interface{} `json:"custom_params,omitempty"`
}

type Module struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Category    string        `json:"category"`
	Params      []ModuleParam `json:"params"`
	Enabled     bool          `json:"enabled"`
}

type ModuleParam struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Label       string      `json:"label"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Options     []string    `json:"options,omitempty"`
	Min         float64     `json:"min,omitempty"`
	Max         float64     `json:"max,omitempty"`
}

type ModuleStatus struct {
	ModuleID      string                 `json:"module_id"`
	Running       bool                   `json:"running"`
	StartTime     *time.Time             `json:"start_time,omitempty"`
	EndTime       *time.Time             `json:"end_time,omitempty"`
	Progress      float64                `json:"progress"`
	CurrentLoad   map[string]interface{} `json:"current_load"`
	Errors        []string               `json:"errors"`
	Results       *LoadResults           `json:"results,omitempty"`
	mu            sync.RWMutex
}

type LoadResults struct {
	TotalRequests   int64                  `json:"total_requests"`
	SuccessCount    int64                  `json:"success_count"`
	ErrorCount      int64                  `json:"error_count"`
	AvgResponseTime float64                `json:"avg_response_time_ms"`
	Throughput      float64                `json:"throughput_rps"`
	Metrics         map[string]interface{} `json:"metrics"`
}

type SystemMetrics struct {
	Timestamp   time.Time   `json:"timestamp"`
	CPU         CPUMetrics  `json:"cpu"`
	Memory      MemMetrics  `json:"memory"`
	Disk        DiskMetrics `json:"disk"`
	Network     NetMetrics  `json:"network"`
	LoadAvg     LoadAvg     `json:"load_avg"`
	Processes   int         `json:"processes"`
}

type CPUMetrics struct {
	UsagePercent float64   `json:"usage_percent"`
	CoreCount    int       `json:"core_count"`
	PerCoreUsage []float64 `json:"per_core_usage"`
}

type MemMetrics struct {
	Total        uint64  `json:"total_bytes"`
	Used         uint64  `json:"used_bytes"`
	Free         uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

type DiskMetrics struct {
	Total        uint64  `json:"total_bytes"`
	Used         uint64  `json:"used_bytes"`
	Free         uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

type NetMetrics struct {
	BytesSent   uint64 `json:"bytes_sent_sec"`
	BytesRecv   uint64 `json:"bytes_recv_sec"`
	Connections int    `json:"active_connections"`
}

type LoadAvg struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

func (ms *ModuleStatus) SetRunning(running bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.Running = running
}

func (ms *ModuleStatus) IsRunning() bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.Running
}

func (ms *ModuleStatus) AddError(err string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.Errors = append(ms.Errors, err)
}
