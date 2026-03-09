package db

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

func InitializeSchema(ctx context.Context, pool *pgxpool.Pool) error {
	// Create extensions
	_, err := pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	if err != nil {
		return err
	}

	// Create tables
	queries := []string{`
		DO $$ 
			BEGIN
				IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'bot_status') THEN
					CREATE TYPE bot_status AS ENUM (
						'UNASSIGNED', 'INITIALIZING', 'RUNNING', 'PAUSED', 'STOPPING', 'STOPPED', 'ERROR', 'RESTARTING'
					);
				END IF;
			END $$;
		`,
		`
		DO $$ 
			BEGIN
				IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'trading_status') THEN
					CREATE TYPE trading_status AS ENUM (
						'IDLE', 'BUYING', 'SELLING', 'HOLDING', 'ANALYZING', 'STOPPED'
					);
				END IF;
			END $$;
		`, `
         DO $$ 
			BEGIN
				IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'side') THEN
					CREATE TYPE side AS ENUM (
						'BUY', 'SELL'
					);
				END IF;
			END $$;
        `, `
         DO $$ 
			BEGIN
				IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'exchange_order_type') THEN
					CREATE TYPE exchange_order_type AS ENUM (
						'MARKET', 'LIMIT'
					);
				END IF;
			END $$;
        `, `
         DO $$ 
			BEGIN
				IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'order_status') THEN
					CREATE TYPE order_status AS ENUM (
						'PENDING', 'FILLED', 'CANCELED'
					);
				END IF;
			END $$;
        `, `
         DO $$ 
			BEGIN
				IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'exchange_order_status') THEN
					CREATE TYPE exchange_order_status AS ENUM (
						'PENDING', 'FILLED', 'CANCELED'
					);
				END IF;
			END $$;
        `, `
         DO $$ 
			BEGIN
				IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'order_type') THEN
					CREATE TYPE order_type AS ENUM (
						'MARKET', 'LIMIT'
					);
				END IF;
			END $$;
        `,
		`CREATE TABLE IF NOT EXISTS bots (
    	 id 					SERIAL PRIMARY KEY,		-- Unique ID for the bot
		 status 				bot_status NOT NULL DEFAULT 'UNASSIGNED',
		 trading_status 		trading_status,
		 pause_reason 			TEXT,
		 paused_at 				TIMESTAMP,
    	 started_at 			TIMESTAMP,
    	 last_heartbeat 		TIMESTAMP,
    	 assigned_process_id	INTEGER,
		 strategy       		VARCHAR(255) 		NOT NULL,	-- Strategy the bot is using (e.g., "moving_average")
		 symbol 				VARCHAR(10) 		NOT NULL,	
		 budget         		NUMERIC(28, 18)     NOT NULL,	-- Budget allocated to the bot
		 total_value            NUMERIC(28, 18)     NOT NULL,	-- Total value of all assets (base + quote) at current market price
		 spent          		NUMERIC(28, 18)		NOT NULL DEFAULT 0.00,					-- Amount spent by the bot
    	 base_asset_balance     NUMERIC(28, 18)		NOT NULL DEFAULT 0.00,					-- Total amount of base asset (e.g., BTC) owned (available_base_asset + reserved_base_asset)
    	 quote_asset_balance    NUMERIC(28, 18)		NOT NULL DEFAULT 0.00,					-- Total amount of quote asset (e.g., USDT) owned (available_quote_asset + reserved_quote_asset)
    	 available_base_asset   NUMERIC(28, 18)		NOT NULL DEFAULT 0.00,					-- Base asset amount available for new sell orders
    	 available_quote_asset  NUMERIC(28, 18)		NOT NULL DEFAULT 0.00,					-- Quote asset amount available for new buy orders
    	 reserved_base_asset    NUMERIC(28, 18)		NOT NULL DEFAULT 0.00,					-- Base asset amount locked in pending sell orders
    	 reserved_quote_asset   NUMERIC(28, 18)		NOT NULL DEFAULT 0.00,					-- Quote asset amount locked in pending buy orders 
		 created_at     		TIMESTAMP WITH TIME ZONE    DEFAULT CURRENT_TIMESTAMP, -- When the bot was created
		 updated_at     		TIMESTAMP WITH TIME ZONE    DEFAULT CURRENT_TIMESTAMP  -- When the bot was last updated
    	);
		`,
		`
		CREATE TABLE IF NOT EXISTS orders (
			id TEXT 		PRIMARY KEY,
			exchange_order_id		VARCHAR(255),
			bot_id 			INTEGER NOT NULL,
			symbol 			VARCHAR(10) NOT NULL,
			side 			side NOT NULL,
			type 			order_type NOT NULL,
			quantity 		NUMERIC(28, 18) NOT NULL,
			price 			NUMERIC(28, 18) NOT NULL,
			status 			order_status NOT NULL,
			filled_qty 		NUMERIC(28, 18) DEFAULT 0,
			filled_price 	NUMERIC(28, 18) DEFAULT 0,
			exchange_id 	TEXT,
			created_at 		TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at 		TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at 	TIMESTAMP,
			FOREIGN KEY (bot_id) REFERENCES bots(id)
		);
     	`, `
		CREATE TABLE IF NOT EXISTS exchange_assets (
			id 				VARCHAR(255) PRIMARY KEY,
			asset 			VARCHAR(10) NOT NULL,
			balance 		NUMERIC(28, 18) NOT NULL,
			created_at 		TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at 		TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
        `, `
		CREATE TABLE IF NOT EXISTS exchange_orders (
			id 				VARCHAR(255) PRIMARY KEY,
		    exchange_id		VARCHAR(255) NOT NULL,
			symbol 			VARCHAR(10) NOT NULL,
			side 			side NOT NULL,
			type 			exchange_order_type NOT NULL,
			status 			exchange_order_status NOT NULL,
			quantity 		NUMERIC(28, 18) NOT NULL,
			price 			NUMERIC(28, 18) NOT NULL,
			filled_qty 		NUMERIC(28, 18) DEFAULT 0.0,
			filled_price 	NUMERIC(28, 18) DEFAULT 0.0,
			created_at 		TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at 		TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			completed_at 	TIMESTAMP WITH TIME ZONE,
			FOREIGN KEY (exchange_id) REFERENCES exchange_assets(id)
		);
        `,
	}

	// Execute all queries in a transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, query := range queries {
		_, err := tx.Exec(ctx, query)

		if err != nil {
			return fmt.Errorf("error: %w", err)
		}
	}

	return tx.Commit(ctx)
}
