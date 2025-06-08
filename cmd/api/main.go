package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lspecian/ovncp/internal/api"
	"github.com/lspecian/ovncp/internal/config"
	"github.com/lspecian/ovncp/internal/db"
	"github.com/lspecian/ovncp/internal/services"
	"github.com/lspecian/ovncp/pkg/ovn"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize logger
	logger, err := initLogger(cfg)
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	// Initialize database
	database, err := db.New(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close()

	// Run migrations
	if err := database.Migrate(); err != nil {
		logger.Fatal("Failed to run database migrations", zap.Error(err))
	}

	// Initialize OVN client
	ovnClient, err := ovn.NewClient(&cfg.OVN)
	if err != nil {
		logger.Fatal("Failed to create OVN client", zap.Error(err))
	}

	// Connect to OVN
	ctx := context.Background()
	if err := ovnClient.Connect(ctx); err != nil {
		logger.Warn("Failed to connect to OVN", zap.Error(err))
		logger.Info("API will start but OVN operations will not be available")
	}
	defer ovnClient.Close()

	// Initialize services
	ovnService := services.NewOVNService(ovnClient)

	// Set up router
	router := api.NewRouter(ovnService, cfg, database, logger)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.API.Host, cfg.API.Port),
		Handler:      router.Engine(),
		ReadTimeout:  cfg.API.ReadTimeout,
		WriteTimeout: cfg.API.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("OVN Control Platform API starting",
			zap.String("host", cfg.API.Host),
			zap.String("port", cfg.API.Port),
			zap.String("environment", cfg.Environment))
		logger.Info("Endpoints available",
			zap.String("health", fmt.Sprintf("http://localhost:%s/health", cfg.API.Port)),
			zap.String("metrics", fmt.Sprintf("http://localhost:%s/metrics", cfg.API.Port)),
			zap.String("api", fmt.Sprintf("http://localhost:%s/api/v1/", cfg.API.Port)))
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func initLogger(cfg *config.Config) (*zap.Logger, error) {
	var zapCfg zap.Config

	if cfg.Environment == "production" {
		zapCfg = zap.NewProductionConfig()
	} else {
		zapCfg = zap.NewDevelopmentConfig()
	}

	// Set log level
	switch cfg.Log.Level {
	case "debug":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	}

	// Set output format
	if cfg.Log.Format == "json" {
		zapCfg.Encoding = "json"
	} else {
		zapCfg.Encoding = "console"
	}

	return zapCfg.Build()
}
