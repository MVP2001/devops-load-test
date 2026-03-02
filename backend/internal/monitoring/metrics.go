package monitoring

import (
	"runtime"
	"sync"
	"time"

	"devops-load-platform/internal/models"
	"devops-load-platform/internal/websocket"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type Metrics struct {
	currentMetrics models.SystemMetrics
	mu             sync.RWMutex
	
	cpuUsage   prometheus.Gauge
	memUsage   prometheus.Gauge
	diskUsage  prometheus.Gauge
	netSent    prometheus.Counter
	netRecv    prometheus.Counter
	goroutines prometheus.Gauge
}

func NewMetrics() *Metrics {
	return &Metrics{
		cpuUsage: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "system_cpu_usage_percent",
			Help: "Current CPU usage percentage",
		}),
		memUsage: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "system_memory_usage_percent",
			Help: "Current memory usage percentage",
		}),
		diskUsage: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "system_disk_usage_percent",
			Help: "Current disk usage percentage",
		}),
		netSent: promauto.NewCounter(prometheus.CounterOpts{
			Name: "system_network_bytes_sent_total",
			Help: "Total network bytes sent",
		}),
		netRecv: promauto.NewCounter(prometheus.CounterOpts{
			Name: "system_network_bytes_recv_total",
			Help: "Total network bytes received",
		}),
		goroutines: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "go_goroutines",
			Help: "Number of goroutines",
		}),
	}
}

func (m *Metrics) StartCollection(hub *websocket.Hub) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastNetIO net.IOCountersStat
	firstRun := true

	for range ticker.C {
		metrics := m.collectMetrics(&lastNetIO, firstRun)
		firstRun = false

		m.mu.Lock()
		m.currentMetrics = metrics
		m.mu.Unlock()

		m.cpuUsage.Set(metrics.CPU.UsagePercent)
		m.memUsage.Set(metrics.Memory.UsagePercent)
		m.diskUsage.Set(metrics.Disk.UsagePercent)
		m.goroutines.Set(float64(runtime.NumGoroutine()))

		if hub != nil {
			hub.BroadcastToChannel("metrics", metrics)
		}
	}
}

func (m *Metrics) collectMetrics(lastNetIO *net.IOCountersStat, firstRun bool) models.SystemMetrics {
	now := time.Now()
	metrics := models.SystemMetrics{
		Timestamp: now,
	}

	if cpuPercents, err := cpu.Percent(0, true); err == nil {
		var total float64
		for _, p := range cpuPercents {
			total += p
		}
		metrics.CPU.UsagePercent = total / float64(len(cpuPercents))
		metrics.CPU.CoreCount = len(cpuPercents)
		metrics.CPU.PerCoreUsage = cpuPercents
	}

	if memInfo, err := mem.VirtualMemory(); err == nil {
		metrics.Memory.Total = memInfo.Total
		metrics.Memory.Used = memInfo.Used
		metrics.Memory.Free = memInfo.Free
		metrics.Memory.UsagePercent = memInfo.UsedPercent
	}

	if diskInfo, err := disk.Usage("/"); err == nil {
		metrics.Disk.Total = diskInfo.Total
		metrics.Disk.Used = diskInfo.Used
		metrics.Disk.Free = diskInfo.Free
		metrics.Disk.UsagePercent = diskInfo.UsedPercent
	}

	if netIO, err := net.IOCounters(false); err == nil && len(netIO) > 0 {
		current := netIO[0]
		if !firstRun {
			metrics.Network.BytesSent = current.BytesSent - lastNetIO.BytesSent
			metrics.Network.BytesRecv = current.BytesRecv - lastNetIO.BytesRecv
		}
		*lastNetIO = current
		m.netSent.Add(float64(metrics.Network.BytesSent))
		m.netRecv.Add(float64(metrics.Network.BytesRecv))
	}

	if conns, err := net.Connections("all"); err == nil {
		metrics.Network.Connections = len(conns)
	}

	if loadAvg, err := load.Avg(); err == nil {
		metrics.LoadAvg.Load1 = loadAvg.Load1
		metrics.LoadAvg.Load5 = loadAvg.Load5
		metrics.LoadAvg.Load15 = loadAvg.Load15
	}

	if procs, err := process.Processes(); err == nil {
		metrics.Processes = len(procs)
	}

	return metrics
}

func (m *Metrics) GetCurrentMetrics() models.SystemMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentMetrics
}
