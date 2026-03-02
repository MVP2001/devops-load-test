package modules

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"devops-load-platform/internal/models"
)

type HTTPFloodModule struct {
	stopChan chan struct{}
	client   *http.Client
}

func NewHTTPFloodModule() *HTTPFloodModule {
	return &HTTPFloodModule{
		stopChan: make(chan struct{}),
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        1000,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  true,
			},
		},
	}
}

func (m *HTTPFloodModule) ID() string { return "http_flood" }
func (m *HTTPFloodModule) Name() string { return "HTTP Flood" }
func (m *HTTPFloodModule) Description() string { 
	return "Массовая генерация HTTP запросов к целевому серверу. Тестирование веб-сервера под нагрузкой." 
}
func (m *HTTPFloodModule) Category() string { return "network" }

func (m *HTTPFloodModule) GetParams() []models.ModuleParam {
	return []models.ModuleParam{
		{
			Name:        "target_url",
			Type:        "string",
			Label:       "Целевой URL",
			Description: "URL для атаки",
			Required:    true,
			Default:     "http://localhost:8080",
		},
		{
			Name:        "method",
			Type:        "select",
			Label:       "HTTP метод",
			Required:    true,
			Default:     "GET",
			Options:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"},
		},
		{
			Name:        "requests_per_second",
			Type:        "number",
			Label:       "RPS",
			Description: "Запросов в секунду",
			Required:    true,
			Default:     1000,
			Min:         1,
			Max:         100000,
		},
		{
			Name:        "payload_size_kb",
			Type:        "number",
			Label:       "Размер payload (KB)",
			Description: "Для POST/PUT запросов",
			Required:    false,
			Default:     0,
			Min:         0,
			Max:         10240,
		},
	}
}

func (m *HTTPFloodModule) ValidateConfig(config models.LoadConfig) error {
	if config.VM.IP == "" {
		return fmt.Errorf("target IP is required")
	}
	return nil
}

func (m *HTTPFloodModule) Start(ctx context.Context, config models.LoadConfig, status *models.ModuleStatus) error {
	targetURL := fmt.Sprintf("http://%s:%d", config.VM.IP, config.VM.Port)
	if url, ok := config.CustomParams["target_url"].(string); ok && url != "" {
		targetURL = url
	}
	
	method := "GET"
	if m, ok := config.CustomParams["method"].(string); ok {
		method = m
	}
	
	rps := 1000
	if r, ok := config.CustomParams["requests_per_second"].(float64); ok {
		rps = int(r)
	}
	
	payloadSize := 0
	if ps, ok := config.CustomParams["payload_size_kb"].(float64); ok {
		payloadSize = int(ps) * 1024
	}

	var requestCount int64
	var successCount int64
	var errorCount int64
	var totalLatency int64
	
	var wg sync.WaitGroup
	workers := config.ConcurrentUsers
	if workers == 0 {
		workers = 100
	}
	
	var payload []byte
	if payloadSize > 0 {
		payload = make([]byte, payloadSize)
		for i := range payload {
			payload[i] = byte(i % 256)
		}
	}
	
	startTime := time.Now()
	
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			ticker := time.NewTicker(time.Second / time.Duration(rps/workers))
			defer ticker.Stop()
			
			for {
				select {
				case <-ctx.Done():
					return
				case <-m.stopChan:
					return
				case <-ticker.C:
					reqStart := time.Now()
					
					var body io.Reader
					if payload != nil {
						body = bytes.NewReader(payload)
					}
					
					req, err := http.NewRequest(method, targetURL, body)
					if err != nil {
						atomic.AddInt64(&errorCount, 1)
						continue
					}
					
					req.Header.Set("User-Agent", fmt.Sprintf("LoadTest/%d", id))
					req.Header.Set("X-Load-Test", "true")
					
					resp, err := m.client.Do(req)
					latency := time.Since(reqStart).Milliseconds()
					
					atomic.AddInt64(&requestCount, 1)
					atomic.AddInt64(&totalLatency, latency)
					
					if err != nil {
						atomic.AddInt64(&errorCount, 1)
						status.AddError(err.Error())
					} else {
						atomic.AddInt64(&successCount, 1)
						resp.Body.Close()
					}
					
					if requestCount%100 == 0 {
						elapsed := time.Since(startTime).Seconds()
						avgLatency := float64(totalLatency) / float64(requestCount)
						currentRPS := float64(requestCount) / elapsed
						
						status.CurrentLoad = map[string]interface{}{
							"requests":    requestCount,
							"success":     successCount,
							"errors":      errorCount,
							"avg_latency": avgLatency,
							"current_rps": currentRPS,
						}
					}
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	elapsed := time.Since(startTime).Seconds()
	status.Results = &models.LoadResults{
		TotalRequests:   requestCount,
		SuccessCount:    successCount,
		ErrorCount:      errorCount,
		AvgResponseTime: float64(totalLatency) / float64(requestCount),
		Throughput:      float64(requestCount) / elapsed,
	}
	
	return nil
}

func (m *HTTPFloodModule) Stop() error {
	close(m.stopChan)
	return nil
}

// TCP SYN Flood Module
type TCPSYNFloodModule struct {
	stopChan chan struct{}
}

func NewTCPSYNFloodModule() *TCPSYNFloodModule {
	return &TCPSYNFloodModule{
		stopChan: make(chan struct{}),
	}
}

func (m *TCPSYNFloodModule) ID() string { return "tcp_syn_flood" }
func (m *TCPSYNFloodModule) Name() string { return "TCP SYN Flood" }
func (m *TCPSYNFloodModule) Description() string { 
	return "TCP SYN flood для тестирования защиты от DDoS" 
}
func (m *TCPSYNFloodModule) Category() string { return "network" }

func (m *TCPSYNFloodModule) GetParams() []models.ModuleParam {
	return []models.ModuleParam{
		{
			Name:        "target_port",
			Type:        "number",
			Label:       "Целевой порт",
			Required:    true,
			Default:     80,
			Min:         1,
			Max:         65535,
		},
		{
			Name:        "packets_per_second",
			Type:        "number",
			Label:       "Пакетов в секунду",
			Required:    true,
			Default:     10000,
			Min:         100,
			Max:         1000000,
		},
	}
}

func (m *TCPSYNFloodModule) ValidateConfig(config models.LoadConfig) error {
	return nil
}

func (m *TCPSYNFloodModule) Start(ctx context.Context, config models.LoadConfig, status *models.ModuleStatus) error {
	target := fmt.Sprintf("%s:%d", config.VM.IP, config.VM.Port)
	if config.VM.Port == 0 {
		target = fmt.Sprintf("%s:80", config.VM.IP)
	}
	
	pps := 10000
	if p, ok := config.CustomParams["packets_per_second"].(float64); ok {
		pps = int(p)
	}

	var packetCount int64
	var successCount int64
	
	ticker := time.NewTicker(time.Second / time.Duration(pps))
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.stopChan:
			return nil
		case <-ticker.C:
			go func() {
				conn, err := net.DialTimeout("tcp", target, 1*time.Second)
				atomic.AddInt64(&packetCount, 1)
				
				if err == nil {
					atomic.AddInt64(&successCount, 1)
					conn.Close()
				}
				
				if packetCount%1000 == 0 {
					status.CurrentLoad = map[string]interface{}{
						"connections_attempted": packetCount,
						"successful":            successCount,
						"target":                target,
					}
				}
			}()
		}
	}
}

func (m *TCPSYNFloodModule) Stop() error {
	close(m.stopChan)
	return nil
}

// UDP Flood Module
type UDPFloodModule struct {
	stopChan chan struct{}
}

func NewUDPFloodModule() *UDPFloodModule {
	return &UDPFloodModule{
		stopChan: make(chan struct{}),
	}
}

func (m *UDPFloodModule) ID() string { return "udp_flood" }
func (m *UDPFloodModule) Name() string { return "UDP Flood" }
func (m *UDPFloodModule) Description() string { 
	return "UDP flood для тестирования UDP сервисов и сетевой инфраструктуры" 
}
func (m *UDPFloodModule) Category() string { return "network" }

func (m *UDPFloodModule) GetParams() []models.ModuleParam {
	return []models.ModuleParam{
		{
			Name:        "target_port",
			Type:        "number",
			Label:       "Целевой порт",
			Required:    true,
			Default:     53,
			Min:         1,
			Max:         65535,
		},
		{
			Name:        "packet_size",
			Type:        "number",
			Label:       "Размер пакета (bytes)",
			Required:    false,
			Default:     1024,
			Min:         64,
			Max:         65507,
		},
		{
			Name:        "packets_per_second",
			Type:        "number",
			Label:       "Пакетов в секунду",
			Required:    true,
			Default:     10000,
			Min:         100,
			Max:         1000000,
		},
	}
}

func (m *UDPFloodModule) ValidateConfig(config models.LoadConfig) error {
	return nil
}

func (m *UDPFloodModule) Start(ctx context.Context, config models.LoadConfig, status *models.ModuleStatus) error {
	target := fmt.Sprintf("%s:%d", config.VM.IP, config.VM.Port)
	if config.VM.Port == 0 {
		target = fmt.Sprintf("%s:53", config.VM.IP)
	}
	
	packetSize := 1024
	if ps, ok := config.CustomParams["packet_size"].(float64); ok {
		packetSize = int(ps)
	}
	
	pps := 10000
	if p, ok := config.CustomParams["packets_per_second"].(float64); ok {
		pps = int(p)
	}

	conn, err := net.Dial("udp", target)
	if err != nil {
		return err
	}
	defer conn.Close()
	
	payload := make([]byte, packetSize)
	for i := range payload {
		payload[i] = byte(i % 256)
	}
	
	var packetCount int64
	ticker := time.NewTicker(time.Second / time.Duration(pps))
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.stopChan:
			return nil
		case <-ticker.C:
			_, err := conn.Write(payload)
			atomic.AddInt64(&packetCount, 1)
			
			if packetCount%10000 == 0 {
				status.CurrentLoad = map[string]interface{}{
					"packets_sent": packetCount,
					"target":       target,
					"errors":       err != nil,
				}
			}
		}
	}
}

func (m *UDPFloodModule) Stop() error {
	close(m.stopChan)
	return nil
}
