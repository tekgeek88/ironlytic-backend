package database

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tekgeek88/ironlytic-backend/internal/config"
)

func InitDatabase(ctx context.Context, cfg config.Config, logger *slog.Logger) (*pgxpool.Pool, error) {
	log := logger.With("component", "db")
	log.Info("Initializing database connection")

	// First connect to postgres database to create our db if it doesn't exist

	adminURL := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort)

	adminPool, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres database: %v", err)
	}
	defer adminPool.Close()

	// Create database if it doesn't exist
	if err := CreateDatabaseIfNotExists(ctx, adminPool, cfg.DBName); err != nil {
		return nil, fmt.Errorf("failed to create database: %v", err)
	}

	// Convert port to int
	dbPort, err := strconv.Atoi(cfg.DBPort)
	if err != nil {
		return nil, fmt.Errorf("invalid port value in DB_PORT: %v", err)
	}

	// Connect to the actual database
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.DBUser, cfg.DBPassword, cfg.DBHost, dbPort, cfg.DBName)

	dbCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, err
	}

	// Set some pool configurations
	dbCfg.MaxConns = 10
	dbCfg.MinConns = 2
	dbCfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, dbCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to application database: %v", err)
	}

	// Initialize schema
	//if err := InitializeSchema(context.Background(), pool); err != nil {
	//	pool.Close()
	//	return nil, fmt.Errorf("failed to initialize schema: %v", err)
	//}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	log.Info("successfully connected to postgres", "db_name", cfg.DBName, "host", cfg.DBHost)
	return pool, nil
}

func CreateDatabaseIfNotExists(ctx context.Context, pool *pgxpool.Pool, dbName string) error {
	var exists int
	err := pool.QueryRow(ctx, `
        SELECT 1 FROM pg_database WHERE datname = $1
    `, dbName).Scan(&exists)

	if err != nil && err.Error() != "no rows in result set" {
		return err
	}

	if exists == 0 {
		_, err = pool.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s`, dbName))
		if err != nil {
			return err
		}
	}

	return nil
}
