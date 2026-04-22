package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/heroiclabs/nakama-common/runtime"
)

func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nakM runtime.NakamaModule, initializer runtime.Initializer) error {
	logger.Info("Initializing Nakama runtime")

	if err := runMigrations(ctx, logger, db); err != nil {
		logger.Error("Failed to run migrations: %v", err)
		return fmt.Errorf("Migration failed: %w", err)
	}

	logger.Info("Registering user metadata update RPC method")
	if err := initializer.RegisterRpc("update_user_metadata", RateLimitMiddleware(UpdateUserMetadata, 5, 10)); err != nil {
		logger.Error("Failed to register update_user_metadata: %v", err)
		return err
	}

	logger.Info("Registering get config RPC method")
	if err := initializer.RegisterRpc("get_game_config", RateLimitMiddleware(GetGameConfig, 20, 30)); err != nil {
		logger.Error("Failed to register get_game_config: %v", err)
		return err
	}

	logger.Info("Registering health check RPC method")
	if err := initializer.RegisterRpc("private_health_check", PrivateHealthCheck); err != nil {
		logger.Error("Failed to register private_health_check: %v", err)
		return err
	}

	logger.Info("All RPCs successfully registered")
	return nil
}

func runMigrations(ctx context.Context, logger runtime.Logger, db *sql.DB) error {
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = 'schema_migrations'
		)
	`).Scan(&exists)

	if err != nil {
		return fmt.Errorf("Failed to check migrations table: %w", err)
	}

	if !exists {
		logger.Info("Creating schema migrations table")
		_, err = db.ExecContext(ctx, `
			CREATE TABLE schema_migrations (
				version VARCHAR(255) PRIMARY KEY,
				applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			)
		`)
		if err != nil {
			return fmt.Errorf("failed to create migrations table: %w", err)
		}
	}

	var migrationApplied bool // maybe enumerate it? something like 0001
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = 'create_game_config')
	`).Scan(&migrationApplied)

	if err != nil {
		return fmt.Errorf("Failed to check migration status: %w", err)
	}

	if !migrationApplied {
		logger.Info("Applying migration create_game_config")

		migrationSQL := `
			CREATE TABLE IF NOT EXISTS game_config (
				id SERIAL PRIMARY KEY,
				config_key VARCHAR(128) NOT NULL UNIQUE,
				config_value JSONB NOT NULL,
				description TEXT,
				is_active BOOLEAN DEFAULT true,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);
			
			CREATE INDEX IF NOT EXISTS idx_game_config_key ON game_config(config_key);
			CREATE INDEX IF NOT EXISTS idx_game_config_active ON game_config(is_active);
			
			INSERT INTO game_config (config_key, config_value, description) VALUES
			('game_settings', '{
				"version": "1.0.0",
				"features": {
					"pvp_enabled": true,
					"chat_enabled": true,
					"guilds_enabled": true,
					"tournaments_enabled": true,
					"daily_rewards": true,
					"maintenance_mode": false
				},
				"economy": {
					"starting_coins": 1000,
					"max_wallet_capacity": 999999,
					"daily_reward_amount": 100,
					"transaction_fee_percent": 5
				},
				"matchmaking": {
					"max_players": 4,
					"min_players": 2,
					"timeout_seconds": 30,
					"skill_tolerance": 200
				},
				"limits": {
					"max_metadata_size_bytes": 16384,
					"max_metadata_keys": 50,
					"metadata_update_rate_seconds": 5
				},
				"announcement": "Welcome to the game!"
			}'::jsonb, 'Core game configuration settings')
			ON CONFLICT (config_key) DO NOTHING;
			
			INSERT INTO schema_migrations (version) VALUES ('create_game_config')
			ON CONFLICT (version) DO NOTHING;
		`

		_, err = db.ExecContext(ctx, migrationSQL)
		if err != nil {
			return fmt.Errorf("Failed to apply migration create_game_config: %w", err)
		}

		logger.Info("Migration create_game_config applied successfully")
	}

	return nil
}
