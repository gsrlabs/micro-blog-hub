package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Database wraps the pgxpool.Pool to provide a unified database access point.
type Database struct {
	Pool *pgxpool.Pool
	//Logger *zap.Logger
}

// Connect establishes a connection pool to PostgreSQL using environment variables
// and automatically executes pending migrations.
func Connect(ctx context.Context, cfg *config.Config) (*Database, error) {

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
		if err := runMigrations(dsn, cfg.Migrations.Path, cfg.App.Mode); err != nil {
			return nil, err
		}
	}
	
	pool, err := pgxpool.NewWithConfig(ctx, pgcfg)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	// Checking the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	log.Printf("INFO: connected to database")

	return &Database{Pool: pool}, nil
}

// runMigrations applies database schema changes using the goose provider
// from the specified migrations directory.
func runMigrations(dsn string, migrationsPath string, mode string) error {

	if mode != "debug" {
        goose.SetLogger(goose.NopLogger()) 
    }

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open sql connection for migrations: %w", err)
	}
	defer func() { _ = db.Close() }()

	//goose.SetBaseFS(os.DirFS("/"))

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	log.Printf("INFO: running migrations from %s", migrationsPath)

	if err := goose.Up(db, migrationsPath); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
