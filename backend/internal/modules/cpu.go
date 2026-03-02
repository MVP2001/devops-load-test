package modules

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"devops-load-platform/internal/models"
)

type CPUBurnModule struct {
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewCPUBurnModule() *CPUBurnModule {
	return &CPUBurnModule{
		stopChan: make(chan struct{}),
	}
}

func (m *CPUBurnModule) ID() string { return "cpu_burn" }
func (m *CPUBurnModule) Name() string { return "CPU Burn" }
func (m *CPUBurnModule) Description() string { 
	return "Максимальная нагрузка на все ядра CPU. Загружает CPU на 100%." 
}
func (m *CPUBurnModule) Category() string { return "cpu" }

func (m *CPUBurnModule) GetParams() []models.ModuleParam {
	return []models.ModuleParam{
		{
			Name:        "cores",
			Type:        "number",
			Label:       "Количество ядер",
			Description: "0 = все ядра",
			Required:    false,
			Default:     0,
			Min:         0,
			Max:         128,
		},
		{
			Name:        "algorithm",
			Type:        "select",
			Label:       "Алгоритм",
			Required:    false,
			Default:     "primes",
			Options:     []string{"primes", "factorial", "fibonacci", "matrix"},
		},
	}
}

func (m *CPUBurnModule) ValidateConfig(config models.LoadConfig) error {
	if config.Duration <= 0 || config.Duration > 1*time.Hour {
		return fmt.Errorf("duration must be between 1s and 1h")
	}
	return nil
}

func (m *CPUBurnModule) Start(ctx context.Context, config models.LoadConfig, status *models.ModuleStatus) error {
	cores := runtime.NumCPU()
	if c, ok := config.CustomParams["cores"].(float64); ok && c > 0 {
		cores = int(c)
	}

	algorithm := "primes"
	if a, ok := config.CustomParams["algorithm"].(string); ok {
		algorithm = a
	}

	var counter int64
	startTime := time.Now()

	for i := 0; i < cores; i++ {
		m.wg.Add(1)
		go func(id int) {
			defer m.wg.Done()
			
			for {
				select {
				case <-ctx.Done():
					return
				case <-m.stopChan:
					return
				default:
					switch algorithm {
					case "primes":
						m.calculatePrimes(id * 1000000)
					case "factorial":
						m.calculateFactorial(10000 + id)
					case "fibonacci":
						m.calculateFibonacci(40)
					case "matrix":
						m.matrixMultiplication()
					}
					atomic.AddInt64(&counter, 1)
					
					elapsed := time.Since(startTime)
					progress := float64(elapsed) / float64(config.Duration) * 100
					if progress > 100 {
						progress = 100
					}
					status.Progress = progress
				}
			}
		}(i)
	}

	m.wg.Wait()
	
	status.Results = &models.LoadResults{
		TotalRequests: counter,
		SuccessCount:  counter,
		Metrics: map[string]interface{}{
			"cores_used": cores,
			"algorithm":  algorithm,
			"operations": counter,
		},
	}
	
	return nil
}

func (m *CPUBurnModule) Stop() error {
	close(m.stopChan)
	return nil
}

func (m *CPUBurnModule) calculatePrimes(start int) {
	for i := start; i < start+10000; i++ {
		isPrime := true
		for j := 2; j*j <= i; j++ {
			if i%j == 0 {
				isPrime = false
				break
			}
		}
		_ = isPrime
	}
}

func (m *CPUBurnModule) calculateFactorial(n int) {
	result := 1.0
	for i := 2; i <= n; i++ {
		result *= float64(i)
		if result > 1e308 {
			result = 1
		}
	}
}

func (m *CPUBurnModule) calculateFibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return m.calculateFibonacci(n-1) + m.calculateFibonacci(n-2)
}

func (m *CPUBurnModule) matrixMultiplication() {
	size := 100
	a := make([][]float64, size)
	b := make([][]float64, size)
	c := make([][]float64, size)
	
	for i := 0; i < size; i++ {
		a[i] = make([]float64, size)
		b[i] = make([]float64, size)
		c[i] = make([]float64, size)
		for j := 0; j < size; j++ {
			a[i][j] = float64(i*j + 1)
			b[i][j] = float64(i + j + 1)
		}
	}
	
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			for k := 0; k < size; k++ {
				c[i][j] += a[i][k] * b[k][j]
			}
		}
	}
}

// Fork Bomb Module
type ForkBombModule struct {
	stopChan chan struct{}
}

func NewForkBombModule() *ForkBombModule {
	return &ForkBombModule{
		stopChan: make(chan struct{}),
	}
}

func (m *ForkBombModule) ID() string { return "fork_bomb" }
func (m *ForkBombModule) Name() string { return "Fork Bomb" }
func (m *ForkBombModule) Description() string { 
	return "⚠️ Создает экспоненциальное количество процессов. Может привести к зависанию!" 
}
func (m *ForkBombModule) Category() string { return "cpu" }

func (m *ForkBombModule) GetParams() []models.ModuleParam {
	return []models.ModuleParam{
		{
			Name:        "max_processes",
			Type:        "number",
			Label:       "Макс. процессов",
			Required:    true,
			Default:     1000,
			Min:         100,
			Max:         10000,
		},
	}
}

func (m *ForkBombModule) ValidateConfig(config models.LoadConfig) error {
	return nil
}

func (m *ForkBombModule) Start(ctx context.Context, config models.LoadConfig, status *models.ModuleStatus) error {
	maxProcs := 1000
	if mp, ok := config.CustomParams["max_processes"].(float64); ok {
		maxProcs = int(mp)
	}

	processCount := 0
	var wg sync.WaitGroup
	
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.stopChan:
			return nil
		default:
			if processCount >= maxProcs {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case <-m.stopChan:
						return
					default:
						_ = math.Sqrt(float64(time.Now().UnixNano()))
					}
				}
			}()
			
			processCount++
			status.CurrentLoad = map[string]interface{}{
				"processes": processCount,
				"max":       maxProcs,
			}
			
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (m *ForkBombModule) Stop() error {
	close(m.stopChan)
	return nil
}
