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

	"github.com/gin-gonic/gin"
	"github.com/gsrlabs/micro-blog-hub/post-service/internal/cache"
	"github.com/gsrlabs/micro-blog-hub/post-service/internal/config"
	"github.com/gsrlabs/micro-blog-hub/post-service/internal/db"
	"github.com/gsrlabs/micro-blog-hub/post-service/internal/handler"
	"github.com/gsrlabs/micro-blog-hub/post-service/internal/logger"
	//"github.com/gsrlabs/micro-blog-hub/post-service/internal/repository"
	"go.uber.org/zap"
)

const configPath = "config/config.yml"

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatalf("application error: %v", err)
	}
}

func run(ctx context.Context) error {

	log.Printf("INFO: starting application")

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("config failed: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config failed: %w", err)
	}

	logger, err := logger.New(cfg.Logging.Level, cfg.App.Mode)
	if err != nil {
		log.Fatalf("%v", err)
		return err
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to sync logger: %v\n", err)
		}
	}()

	//Mongo
	database, err := db.NewMongoCLient(ctx,
		logger,
		cfg.Mongo.Host,
		cfg.Mongo.Port,
	)

	if err != nil {
		log.Fatalf("Mongo connection failed: %v", err)
	}

	defer func() {
		if err := database.Disconnect(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "failed to disconnect mongodb: %v\n", err)
		}
	}()

	//Redis
	redisClient, err := cache.NewRedisClient(
		ctx,
		logger,
		cfg.Redis.Host,
		cfg.Redis.Port,
	)
	if err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	defer func() {
		if err := redisClient.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to disconnect redis: %v\n", err)
		}
	}()

	// Repository
	//postRepo := repository.NewPostRepository(database, cfg.Mongo.DB, logger)

	//HTTP
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(handler.ZapLogger(logger))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	server := &http.Server{
		Addr:    ":" + cfg.App.Port,
		Handler: r,
	}

	go func() {
		logger.Info("HTTP server started", zap.String("URL", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server listen error", zap.Error(err))

		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctxShoutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctxShoutdown); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	logger.Info("Server exiting")

	return nil
}
