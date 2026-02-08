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
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/config"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/db"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/handler"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/logger"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/repository"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/service"
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
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	logger, err := logger.New(cfg.Logging.Level, cfg.App.Mode)
	if err != nil {
		return err
	}
	defer logger.Sync()

	// 1️⃣ DB
	database, err := db.Connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer database.Pool.Close()

	// 2️⃣ Repository
	authRepo := repository.NewAuthRepository(database.Pool, logger)

	// 3️⃣ Service
	authService := service.NewAuthService(authRepo, logger)

	// 4️⃣ Handler
	h := handler.NewAuthHandler(authService, logger)

	// Устанавливаем режим работы Gin
    if cfg.App.Mode == "release" {
        gin.SetMode(gin.ReleaseMode)
    } else {
        gin.SetMode(gin.DebugMode)
    }

	// 5️⃣ Router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(handler.ZapLogger(logger))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	auth := r.Group("/auth")
	auth.POST("", h.Create)

	user := r.Group("/user")
	user.GET("/:id", h.GetByID)
	user.GET("/search", h.GetByEmail)
	//user.PUT("/:id", h.Update)
	//user.DELETE("/:id", h.Delete)
	//user.GET("", h.List)

	server := &http.Server{
		Addr:    ":" + cfg.App.Port,
		Handler: r,
	}

	go func() {
		log.Printf("INFO: HTTP server started on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen: %s\n", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    logger.Info("Shutting down server...")

    ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := server.Shutdown(ctxShutdown); err != nil {
        return fmt.Errorf("server forced to shutdown: %w", err)
    }

    logger.Info("Server exiting")

	return nil
}

