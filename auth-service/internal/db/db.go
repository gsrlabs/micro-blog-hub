package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

// Database wraps the pgxpool.Pool to provide a unified database access point.
type Database struct {
	Pool   *pgxpool.Pool
	Logger *zap.Logger
}

var (
	newPoolWithConfig = pgxpool.NewWithConfig
	sqlOpen           = sql.Open
	gooseUp           = goose.Up
	gooseSetDialect   = goose.SetDialect
)

// Connect establishes a connection pool to PostgreSQL using environment variables
// and automatically executes pending migrations.
func Connect(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*Database, error) {

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	pgcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pgx config: %w", err)
	}

	pgcfg.MaxConns = cfg.Database.MaxConns
	pgcfg.MinConns = cfg.Database.MinConns
	pgcfg.MaxConnLifetime = time.Hour

	if cfg.Migrations.Auto {
		if err := runMigrations(dsn, cfg.Migrations.Path, cfg.App.Mode, logger); err != nil {
			return nil, err
		}
	}

	pool, err := newPoolWithConfig(ctx, pgcfg)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	logger.Info("connected to database")
	return &Database{Pool: pool, Logger: logger}, nil
}

// runMigrations applies database schema changes using the goose provider
// from the specified migrations directory.
func runMigrations(dsn, migrationsPath, mode string, logger *zap.Logger) error {
	if mode != "debug" {
		goose.SetLogger(goose.NopLogger())
	}

	db, err := sqlOpen("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open sql connection for migrations: %w", err)
	}
	defer db.Close()

	if err := gooseSetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	logger.Info("running migrations", zap.String("path", migrationsPath))

	if err := gooseUp(db, migrationsPath); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	logger.Info("migrations finished successfully")
	return nil
}
