package modules

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"devops-load-platform/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

// DB Connection Flood Module
type DBConnectionFloodModule struct {
	stopChan chan struct{}
}

func NewDBConnectionFloodModule() *DBConnectionFloodModule {
	return &DBConnectionFloodModule{
		stopChan: make(chan struct{}),
	}
}

func (m *DBConnectionFloodModule) ID() string { return "db_flood" }
func (m *DBConnectionFloodModule) Name() string { return "DB Connection Flood" }
func (m *DBConnectionFloodModule) Description() string { 
	return "Создание множества соединений с БД и выполнение тяжелых запросов. Тестирует connection pool." 
}
func (m *DBConnectionFloodModule) Category() string { return "application" }

func (m *DBConnectionFloodModule) GetParams() []models.ModuleParam {
	return []models.ModuleParam{
		{
			Name:        "db_type",
			Type:        "select",
			Label:       "Тип БД",
			Required:    true,
			Default:     "sqlite",
			Options:     []string{"sqlite", "mysql", "postgres"},
		},
		{
			Name:        "connection_string",
			Type:        "string",
			Label:       "Строка подключения",
			Description: "DSN для подключения к БД",
			Required:    false,
			Default:     ":memory:",
		},
		{
			Name:        "max_connections",
			Type:        "number",
			Label:       "Макс. соединений",
			Required:    true,
			Default:     100,
			Min:         10,
			Max:         10000,
		},
		{
			Name:        "query_complexity",
			Type:        "select",
			Label:       "Сложность запросов",
			Required:    false,
			Default:     "medium",
			Options:     []string{"simple", "medium", "complex", "extreme"},
		},
	}
}

func (m *DBConnectionFloodModule) ValidateConfig(config models.LoadConfig) error {
	return nil
}

func (m *DBConnectionFloodModule) Start(ctx context.Context, config models.LoadConfig, status *models.ModuleStatus) error {
	dbType := "sqlite"
	if dt, ok := config.CustomParams["db_type"].(string); ok {
		dbType = dt
	}
	
	connStr := ":memory:"
	if cs, ok := config.CustomParams["connection_string"].(string); ok && cs != "" {
		connStr = cs
	}
	
	maxConns := 100
	if mc, ok := config.CustomParams["max_connections"].(float64); ok {
		maxConns = int(mc)
	}
	
	complexity := "medium"
	if c, ok := config.CustomParams["query_complexity"].(string); ok {
		complexity = c
	}

	var driver string
	switch dbType {
	case "sqlite":
		driver = "sqlite3"
	case "mysql":
		driver = "mysql"
	case "postgres":
		driver = "postgres"
	}
	
	db, err := sql.Open(driver, connStr)
	if err != nil {
		return err
	}
	defer db.Close()
	
	db.Exec(`CREATE TABLE IF NOT EXISTS load_test (
		id INTEGER PRIMARY KEY,
		data TEXT,
		created_at TIMESTAMP,
		value REAL
	)`)
	
	var wg sync.WaitGroup
	var queryCount int64
	var errorCount int64
	
	for i := 0; i < maxConns; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			localDB, err := sql.Open(driver, connStr)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				return
			}
			defer localDB.Close()
			
			for {
				select {
				case <-ctx.Done():
					return
				case <-m.stopChan:
					return
				default:
					var query string
					switch complexity {
					case "simple":
						query = "SELECT 1"
					case "medium":
						query = fmt.Sprintf("INSERT INTO load_test (data, created_at, value) VALUES ('test_%d', datetime('now'), %f)", 
							id, float64(id)*3.14)
					case "complex":
						query = fmt.Sprintf(`
							SELECT t1.id, t2.id, COUNT(*) 
							FROM load_test t1 
							CROSS JOIN load_test t2 
							WHERE t1.id > %d 
							GROUP BY t1.id, t2.id 
							ORDER BY t1.id`, id)
					case "extreme":
						query = `
							WITH RECURSIVE r(n) AS (
								SELECT 1
								UNION ALL
								SELECT n + 1 FROM r WHERE n < 10000
							)
							SELECT count(*) FROM r`
					}
					
					_, err := localDB.Exec(query)
					atomic.AddInt64(&queryCount, 1)
					
					if err != nil {
						atomic.AddInt64(&errorCount, 1)
					}
					
					if queryCount%100 == 0 {
						status.CurrentLoad = map[string]interface{}{
							"queries_executed": queryCount,
							"errors":           errorCount,
							"active_workers":   maxConns,
							"complexity":       complexity,
						}
					}
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	status.Results = &models.LoadResults{
		TotalRequests: queryCount,
		SuccessCount:  queryCount - errorCount,
		ErrorCount:    errorCount,
		Metrics: map[string]interface{}{
			"db_type":          dbType,
			"max_connections":  maxConns,
			"query_complexity": complexity,
		},
	}
	
	return nil
}

func (m *DBConnectionFloodModule) Stop() error {
	close(m.stopChan)
	return nil
}

// Log Flood Module
type LogFloodModule struct {
	stopChan chan struct{}
}

func NewLogFloodModule() *LogFloodModule {
	return &LogFloodModule{
		stopChan: make(chan struct{}),
	}
}

func (m *LogFloodModule) ID() string { return "log_flood" }
func (m *LogFloodModule) Name() string { return "Log Flood" }
func (m *LogFloodModule) Description() string { 
	return "Генерация миллионов лог-сообщений для тестирования систем сбора логов (ELK, Loki, etc.)" 
}
func (m *LogFloodModule) Category() string { return "application" }

func (m *LogFloodModule) GetParams() []models.ModuleParam {
	return []models.ModuleParam{
		{
			Name:        "log_file",
			Type:        "string",
			Label:       "Файл логов",
			Description: "Путь к файлу для записи логов",
			Required:    true,
			Default:     "/tmp/loadtest.log",
		},
		{
			Name:        "logs_per_second",
			Type:        "number",
			Label:       "Логов в секунду",
			Required:    true,
			Default:     10000,
			Min:         100,
			Max:         1000000,
		},
		{
			Name:        "log_level",
			Type:        "select",
			Label:       "Уровень логов",
			Required:    false,
			Default:     "INFO",
			Options:     []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"},
		},
		{
			Name:        "include_stacktrace",
			Type:        "boolean",
			Label:       "Включать stacktrace",
			Description: "Добавлять большие stacktrace к логам",
			Required:    false,
			Default:     false,
		},
	}
}

func (m *LogFloodModule) ValidateConfig(config models.LoadConfig) error {
	return nil
}

func (m *LogFloodModule) Start(ctx context.Context, config models.LoadConfig, status *models.ModuleStatus) error {
	logFile := "/tmp/loadtest.log"
	if lf, ok := config.CustomParams["log_file"].(string); ok && lf != "" {
		logFile = lf
	}
	
	logsPerSecond := 10000
	if lps, ok := config.CustomParams["logs_per_second"].(float64); ok {
		logsPerSecond = int(lps)
	}
	
	logLevel := "INFO"
	if ll, ok := config.CustomParams["log_level"].(string); ok {
		logLevel = ll
	}
	
	includeStacktrace := false
	if is, ok := config.CustomParams["include_stacktrace"].(bool); ok {
		includeStacktrace = is
	}

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	var logCount int64
	var bytesWritten int64
	
	ticker := time.NewTicker(time.Second / time.Duration(logsPerSecond))
	defer ticker.Stop()
	
	stacktrace := ""
	if includeStacktrace {
		for i := 0; i < 100; i++ {
			stacktrace += fmt.Sprintf("at com.example.Module%d.method%d(Line:%d)\n", i, i*2, i*10)
		}
	}
	
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.stopChan:
			return nil
		case <-ticker.C:
			timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
			message := fmt.Sprintf("[%s] %s - Load test log message %d from module %s", 
				timestamp, logLevel, logCount, m.ID())
			
			if includeStacktrace {
				message += "\nStacktrace:\n" + stacktrace
			}
			
			message += "\n"
			
			n, err := file.WriteString(message)
			if err != nil {
				status.AddError(err.Error())
				continue
			}
			
			atomic.AddInt64(&logCount, 1)
			atomic.AddInt64(&bytesWritten, int64(n))
			
			if logCount%1000 == 0 {
				file.Sync()
				
				status.CurrentLoad = map[string]interface{}{
					"logs_written":  logCount,
					"bytes_written": bytesWritten,
					"file_size_mb":  bytesWritten / 1024 / 1024,
					"log_level":     logLevel,
				}
			}
		}
	}
}

func (m *LogFloodModule) Stop() error {
	close(m.stopChan)
	return nil
}
