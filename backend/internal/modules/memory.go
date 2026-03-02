package modules

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"devops-load-platform/internal/models"
)

type MemoryFillModule struct {
	stopChan chan struct{}
	data     [][]byte
	mu       sync.Mutex
}

func NewMemoryFillModule() *MemoryFillModule {
	return &MemoryFillModule{
		stopChan: make(chan struct{}),
		data:     make([][]byte, 0),
	}
}

func (m *MemoryFillModule) ID() string { return "memory_fill" }
func (m *MemoryFillModule) Name() string { return "Memory Fill" }
func (m *MemoryFillModule) Description() string { 
	return "Постепенное заполнение оперативной памяти. Проверка поведения при OOM." 
}
func (m *MemoryFillModule) Category() string { return "memory" }

func (m *MemoryFillModule) GetParams() []models.ModuleParam {
	return []models.ModuleParam{
		{
			Name:        "target_percent",
			Type:        "number",
			Label:       "Целевой % заполнения",
			Required:    true,
			Default:     90,
			Min:         10,
			Max:         99,
		},
		{
			Name:        "chunk_size_mb",
			Type:        "number",
			Label:       "Размер чанка (MB)",
			Required:    false,
			Default:     100,
			Min:         10,
			Max:         1000,
		},
	}
}

func (m *MemoryFillModule) ValidateConfig(config models.LoadConfig) error {
	return nil
}

func (m *MemoryFillModule) Start(ctx context.Context, config models.LoadConfig, status *models.ModuleStatus) error {
	targetPercent := 90.0
	if tp, ok := config.CustomParams["target_percent"].(float64); ok {
		targetPercent = tp
	}
	
	chunkSize := 100 * 1024 * 1024
	if cs, ok := config.CustomParams["chunk_size_mb"].(float64); ok {
		chunkSize = int(cs) * 1024 * 1024
	}

	var mStats runtime.MemStats
	runtime.ReadMemStats(&mStats)
	totalMemory := mStats.Sys
	
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.stopChan:
			return nil
		case <-ticker.C:
			runtime.ReadMemStats(&mStats)
			currentPercent := float64(mStats.Alloc) / float64(totalMemory) * 100
			
			status.Progress = currentPercent / targetPercent * 100
			status.CurrentLoad = map[string]interface{}{
				"allocated_mb":    mStats.Alloc / 1024 / 1024,
				"system_mb":       mStats.Sys / 1024 / 1024,
				"target_percent":  targetPercent,
				"current_percent": currentPercent,
			}
			
			if currentPercent >= targetPercent {
				continue
			}
			
			data := make([]byte, chunkSize)
			for i := range data {
				data[i] = byte(i % 256)
			}
			
			m.mu.Lock()
			m.data = append(m.data, data)
			m.mu.Unlock()
		}
	}
}

func (m *MemoryFillModule) Stop() error {
	close(m.stopChan)
	m.mu.Lock()
	m.data = nil
	m.mu.Unlock()
	runtime.GC()
	return nil
}

// Memory Leak Module
type MemoryLeakModule struct {
	stopChan chan struct{}
	leaked   map[string][]byte
	mu       sync.Mutex
}

func NewMemoryLeakModule() *MemoryLeakModule {
	return &MemoryLeakModule{
		stopChan: make(chan struct{}),
		leaked:   make(map[string][]byte),
	}
}

func (m *MemoryLeakModule) ID() string { return "memory_leak" }
func (m *MemoryLeakModule) Name() string { return "Memory Leak Simulator" }
func (m *MemoryLeakModule) Description() string { 
	return "Симуляция утечки памяти для анализа профилировщиками." 
}
func (m *MemoryLeakModule) Category() string { return "memory" }

func (m *MemoryLeakModule) GetParams() []models.ModuleParam {
	return []models.ModuleParam{
		{
			Name:        "leak_rate_mb",
			Type:        "number",
			Label:       "Скорость утечки (MB/s)",
			Required:    true,
			Default:     10,
			Min:         1,
			Max:         1000,
		},
		{
			Name:        "leak_type",
			Type:        "select",
			Label:       "Тип утечки",
			Required:    false,
			Default:     "linear",
			Options:     []string{"linear", "exponential", "burst"},
		},
	}
}

func (m *MemoryLeakModule) ValidateConfig(config models.LoadConfig) error {
	return nil
}

func (m *MemoryLeakModule) Start(ctx context.Context, config models.LoadConfig, status *models.ModuleStatus) error {
	leakRate := 10
	if lr, ok := config.CustomParams["leak_rate_mb"].(float64); ok {
		leakRate = int(lr)
	}
	
	leakType := "linear"
	if lt, ok := config.CustomParams["leak_type"].(string); ok {
		leakType = lt
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	iteration := 0
	
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.stopChan:
			return nil
		case <-ticker.C:
			iteration++
			
			var size int
			switch leakType {
			case "linear":
				size = leakRate * 1024 * 1024
			case "exponential":
				size = leakRate * 1024 * 1024 * iteration
			case "burst":
				if iteration%5 == 0 {
					size = leakRate * 5 * 1024 * 1024
				} else {
					size = leakRate * 1024 * 1024 / 2
				}
			}
			
			key := fmt.Sprintf("leak_%d_%d", time.Now().Unix(), iteration)
			data := make([]byte, size)
			
			for i := range data {
				data[i] = byte(i % 256)
			}
			
			m.mu.Lock()
			m.leaked[key] = data
			totalLeaked := len(m.leaked) * size
			m.mu.Unlock()
			
			var mStats runtime.MemStats
			runtime.ReadMemStats(&mStats)
			
			status.CurrentLoad = map[string]interface{}{
				"leaked_objects":  len(m.leaked),
				"total_leaked_mb": totalLeaked / 1024 / 1024,
				"heap_alloc_mb":   mStats.HeapAlloc / 1024 / 1024,
				"iteration":       iteration,
			}
		}
	}
}

func (m *MemoryLeakModule) Stop() error {
	close(m.stopChan)
	m.mu.Lock()
	m.leaked = make(map[string][]byte)
	m.mu.Unlock()
	runtime.GC()
	return nil
}
