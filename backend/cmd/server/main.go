package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"devops-load-platform/internal/monitoring"
	"devops-load-platform/internal/modules"
	"devops-load-platform/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	r := gin.Default()
	
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	hub := websocket.NewHub()
	go hub.Run()

	metrics := monitoring.NewMetrics()
	go metrics.StartCollection(hub)

	api := r.Group("/api/v1")
	{
		loadModules := modules.NewManager(hub, metrics, logger)
		
		api.GET("/modules", loadModules.GetModules)
		api.POST("/modules/:id/start", loadModules.StartModule)
		api.POST("/modules/:id/stop", loadModules.StopModule)
		api.GET("/modules/:id/status", loadModules.GetStatus)
		
		api.GET("/metrics/realtime", func(c *gin.Context) {
			c.JSON(200, metrics.GetCurrentMetrics())
		})
	}

	r.GET("/ws", func(c *gin.Context) {
		websocket.ServeWs(hub, c.Writer, c.Request)
	})

	r.GET("/prometheus", gin.WrapH(promhttp.Handler()))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "timestamp": time.Now()})
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	logger.Info("Server started on :8080")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}
}
