package modules

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"devops-load-platform/internal/models"
	"devops-load-platform/internal/monitoring"
	"devops-load-platform/internal/websocket"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type LoadModule interface {
	ID() string
	Name() string
	Description() string
	Category() string
	GetParams() []models.ModuleParam
	ValidateConfig(config models.LoadConfig) error
	Start(ctx context.Context, config models.LoadConfig, status *models.ModuleStatus) error
	Stop() error
}

type Manager struct {
	modules  map[string]LoadModule
	statuses map[string]*models.ModuleStatus
	hub      *websocket.Hub
	metrics  *monitoring.Metrics
	logger   *zap.Logger
	mu       sync.RWMutex
}

func NewManager(hub *websocket.Hub, metrics *monitoring.Metrics, logger *zap.Logger) *Manager {
	m := &Manager{
		modules:  make(map[string]LoadModule),
		statuses: make(map[string]*models.ModuleStatus),
		hub:      hub,
		metrics:  metrics,
		logger:   logger,
	}
	m.registerModules()
	return m
}

func (m *Manager) registerModules() {
	// CPU
	m.Register(NewCPUBurnModule())
	m.Register(NewForkBombModule())
	// Memory
	m.Register(NewMemoryFillModule())
	m.Register(NewMemoryLeakModule())
	// Disk
	m.Register(NewDiskFillModule())
	m.Register(NewDiskIOModule())
	// Network
	m.Register(NewHTTPFloodModule())
	m.Register(NewTCPSYNFloodModule())
	m.Register(NewUDPFloodModule())
	// Application
	m.Register(NewDBConnectionFloodModule())
	m.Register(NewLogFloodModule())
	// Security
	m.Register(NewBruteForceModule())
	m.Register(NewPathTraversalModule())
	m.Register(NewSQLInjectionModule())
}

func (m *Manager) Register(module LoadModule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.modules[module.ID()] = module
	m.statuses[module.ID()] = &models.ModuleStatus{
		ModuleID: module.ID(),
		Running:  false,
		Errors:   []string{},
	}
}

func (m *Manager) GetModules(c *gin.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var modules []models.Module
	for _, mod := range m.modules {
		modules = append(modules, models.Module{
			ID:          mod.ID(),
			Name:        mod.Name(),
			Description: mod.Description(),
			Category:    mod.Category(),
			Params:      mod.GetParams(),
			Enabled:     true,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"modules": modules,
		"count":   len(modules),
	})
}

func (m *Manager) StartModule(c *gin.Context) {
	moduleID := c.Param("id")
	
	m.mu.RLock()
	module, exists := m.modules[moduleID]
	status := m.statuses[moduleID]
	m.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
		return
	}

	if status.IsRunning() {
		c.JSON(http.StatusConflict, gin.H{"error": "Module already running"})
		return
	}

	var config models.LoadConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := module.ValidateConfig(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status.SetRunning(true)
	status.StartTime = &[]time.Time{time.Now()}[0]
	status.EndTime = nil
	status.Progress = 0
	status.Errors = []string{}
	status.Results = nil

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	
	go func() {
		defer cancel()
		m.broadcastStatus(moduleID, status)
		
		err := module.Start(ctx, config, status)
		
		status.SetRunning(false)
		now := time.Now()
		status.EndTime = &now
		
		if err != nil {
			status.AddError(err.Error())
			m.logger.Error("Module execution failed", zap.String("module", moduleID), zap.Error(err))
		}
		
		m.broadcastStatus(moduleID, status)
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Module started",
		"module":  moduleID,
		"status":  status,
	})
}

func (m *Manager) StopModule(c *gin.Context) {
	moduleID := c.Param("id")
	
	m.mu.RLock()
	module, exists := m.modules[moduleID]
	status := m.statuses[moduleID]
	m.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
		return
	}

	if !status.IsRunning() {
		c.JSON(http.StatusConflict, gin.H{"error": "Module not running"})
		return
	}

	if err := module.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	status.SetRunning(false)
	now := time.Now()
	status.EndTime = &now
	
	m.broadcastStatus(moduleID, status)

	c.JSON(http.StatusOK, gin.H{
		"message": "Module stopped",
		"module":  moduleID,
	})
}

func (m *Manager) GetStatus(c *gin.Context) {
	moduleID := c.Param("id")
	
	m.mu.RLock()
	_, exists := m.modules[moduleID]
	status := m.statuses[moduleID]
	m.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
		return
	}

	c.JSON(http.StatusOK, status)
}

func (m *Manager) broadcastStatus(moduleID string, status *models.ModuleStatus) {
	if m.hub != nil {
		m.hub.BroadcastToChannel(fmt.Sprintf("module_%s", moduleID), status)
		m.hub.BroadcastToChannel("all_modules", map[string]interface{}{
			"module_id": moduleID,
			"status":    status,
		})
	}
}
