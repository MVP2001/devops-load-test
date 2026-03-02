package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"devops-load-platform/internal/models"
)

type DiskFillModule struct {
	stopChan chan struct{}
	files    []string
	mu       sync.Mutex
}

func NewDiskFillModule() *DiskFillModule {
	return &DiskFillModule{
		stopChan: make(chan struct{}),
		files:    make([]string, 0),
	}
}

func (m *DiskFillModule) ID() string { return "disk_fill" }
func (m *DiskFillModule) Name() string { return "Disk Fill" }
func (m *DiskFillModule) Description() string { 
	return "Заполнение дискового пространства файлами. Проверка обработки ошибок при нехватке места." 
}
func (m *DiskFillModule) Category() string { return "disk" }

func (m *DiskFillModule) GetParams() []models.ModuleParam {
	return []models.ModuleParam{
		{
			Name:        "target_path",
			Type:        "string",
			Label:       "Целевая директория",
			Description: "Путь для создания файлов",
			Required:    true,
			Default:     "/tmp/loadtest",
		},
		{
			Name:        "target_percent",
			Type:        "number",
			Label:       "Целевой % заполнения",
			Required:    true,
			Default:     95,
			Min:         50,
			Max:         99,
		},
		{
			Name:        "file_size_mb",
			Type:        "number",
			Label:       "Размер файла (MB)",
			Required:    false,
			Default:     100,
			Min:         1,
			Max:         1000,
		},
	}
}

func (m *DiskFillModule) ValidateConfig(config models.LoadConfig) error {
	return nil
}

func (m *DiskFillModule) Start(ctx context.Context, config models.LoadConfig, status *models.ModuleStatus) error {
	targetPath := "/tmp/loadtest"
	if tp, ok := config.CustomParams["target_path"].(string); ok {
		targetPath = tp
	}
	
	targetPercent := 95.0
	if tp, ok := config.CustomParams["target_percent"].(float64); ok {
		targetPercent = tp
	}
	
	fileSizeMB := 100
	if fs, ok := config.CustomParams["file_size_mb"].(float64); ok {
		fileSizeMB = int(fs)
	}

	os.MkdirAll(targetPath, 0755)
	
	fileCount := 0
	fileSize := fileSizeMB * 1024 * 1024
	
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.stopChan:
			return nil
		case <-ticker.C:
			var stat syscall.Statfs_t
			syscall.Statfs(targetPath, &stat)
			
			free := stat.Bavail * uint64(stat.Bsize)
			total := stat.Blocks * uint64(stat.Bsize)
			used := total - free
			usedPercent := float64(used) / float64(total) * 100
			
			status.Progress = usedPercent / targetPercent * 100
			status.CurrentLoad = map[string]interface{}{
				"files_created":  fileCount,
				"used_percent":   usedPercent,
				"free_gb":        free / 1024 / 1024 / 1024,
				"target_percent": targetPercent,
			}
			
			if usedPercent >= targetPercent {
				continue
			}
			
			filename := filepath.Join(targetPath, fmt.Sprintf("loadtest_%d_%d.bin", time.Now().Unix(), fileCount))
			data := make([]byte, fileSize)
			
			for i := range data {
				data[i] = byte(i % 256)
			}
			
			err := os.WriteFile(filename, data, 0644)
			if err != nil {
				status.AddError(fmt.Sprintf("Failed to write file: %v", err))
				continue
			}
			
			m.mu.Lock()
			m.files = append(m.files, filename)
			m.mu.Unlock()
			
			fileCount++
		}
	}
}

func (m *DiskFillModule) Stop() error {
	close(m.stopChan)
	m.mu.Lock()
	for _, file := range m.files {
		os.Remove(file)
	}
	m.files = make([]string, 0)
	m.mu.Unlock()
	return nil
}

// Disk IO Module
type DiskIOModule struct {
	stopChan chan struct{}
}

func NewDiskIOModule() *DiskIOModule {
	return &DiskIOModule{
		stopChan: make(chan struct{}),
	}
}

func (m *DiskIOModule) ID() string { return "disk_io" }
func (m *DiskIOModule) Name() string { return "Disk IO Stress" }
func (m *DiskIOModule) Description() string { 
	return "Интенсивные операции чтения/записи для создания дисковой нагрузки" 
}
func (m *DiskIOModule) Category() string { return "disk" }

func (m *DiskIOModule) GetParams() []models.ModuleParam {
	return []models.ModuleParam{
		{
			Name:        "io_pattern",
			Type:        "select",
			Label:       "Паттерн IO",
			Description: "Тип дисковых операций",
			Required:    true,
			Default:     "mixed",
			Options:     []string{"sequential_read", "sequential_write", "random_read", "random_write", "mixed"},
		},
		{
			Name:        "block_size_kb",
			Type:        "number",
			Label:       "Размер блока (KB)",
			Required:    false,
			Default:     4,
			Min:         1,
			Max:         4096,
		},
		{
			Name:        "queue_depth",
			Type:        "number",
			Label:       "Глубина очереди",
			Description: "Количество параллельных операций",
			Required:    false,
			Default:     32,
			Min:         1,
			Max:         256,
		},
	}
}

func (m *DiskIOModule) ValidateConfig(config models.LoadConfig) error {
	return nil
}

func (m *DiskIOModule) Start(ctx context.Context, config models.LoadConfig, status *models.ModuleStatus) error {
	ioPattern := "mixed"
	if ip, ok := config.CustomParams["io_pattern"].(string); ok {
		ioPattern = ip
	}
	
	blockSize := 4 * 1024
	if bs, ok := config.CustomParams["block_size_kb"].(float64); ok {
		blockSize = int(bs) * 1024
	}
	
	queueDepth := 32
	if qd, ok := config.CustomParams["queue_depth"].(float64); ok {
		queueDepth = int(qd)
	}

	tempFile := "/tmp/io_stress_test.dat"
	file, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	defer os.Remove(tempFile)
	defer file.Close()
	
	data := make([]byte, 1024*1024)
	for i := 0; i < 1024; i++ {
		file.Write(data)
	}
	file.Sync()
	
	var wg sync.WaitGroup
	opsCount := 0
	var mu sync.Mutex
	
	for i := 0; i < queueDepth; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			localFile, _ := os.OpenFile(tempFile, os.O_RDWR, 0644)
			if localFile == nil {
				return
			}
			defer localFile.Close()
			
			buffer := make([]byte, blockSize)
			
			for {
				select {
				case <-ctx.Done():
					return
				case <-m.stopChan:
					return
				default:
					switch ioPattern {
					case "sequential_read":
						localFile.Read(buffer)
					case "sequential_write":
						localFile.Write(buffer)
					case "random_read":
						offset := int64(id * blockSize)
						localFile.ReadAt(buffer, offset)
					case "random_write":
						offset := int64(id * blockSize)
						localFile.WriteAt(buffer, offset)
					case "mixed":
						if opsCount%2 == 0 {
							localFile.Read(buffer)
						} else {
							localFile.Write(buffer)
						}
					}
					
					mu.Lock()
					opsCount++
					mu.Unlock()
					
					if opsCount%1000 == 0 {
						status.CurrentLoad = map[string]interface{}{
							"ops_per_sec": opsCount,
							"pattern":     ioPattern,
							"workers":     queueDepth,
						}
					}
				}
			}
		}(i)
	}
	
	wg.Wait()
	return nil
}

func (m *DiskIOModule) Stop() error {
	close(m.stopChan)
	return nil
}
