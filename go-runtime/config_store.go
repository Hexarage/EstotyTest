package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

var (
	configCache     map[string]GameConfig
	configCacheMu   sync.RWMutex
	configCacheTTL  = 30 * time.Second
	lastCacheUpdate time.Time
)

type GameConfig struct {
	Version        string                 `json:"version"`
	ServerTime     int64                  `json:"server_time"`
	Features       map[string]bool        `json:"features"`
	Economy        EconomyConfig          `json:"economy"`
	Matchmaking    MatchmakingConfig      `json:"matchmaking"`
	Limits         LimitsConfig           `json:"limits"`
	CustomSettings map[string]interface{} `json:"custom_settings"`
	Announcement   string                 `json:"announcement"`
}

type EconomyConfig struct {
	StartingCoins         int64 `json:"starting_coins"`
	MaxWalletCapacity     int64 `json:"max_wallet_capacity"`
	DailyRewardAmount     int64 `json:"daily_reward_amount"`
	TransactionFeePercent int   `json:"transaction_fee_percent"`
}

type MatchmakingConfig struct {
	MaxPlayers     int `json:"max_players"`
	MinPlayers     int `json:"min_players"`
	TimeoutSeconds int `json:"timeout_seconds"`
	SkillTolerance int `json:"skill_tolerance"`
}

type LimitsConfig struct {
	MaxMetadataSizeBytes      int `json:"max_metadata_size_bytes"`
	MaxMetadataKeys           int `json:"max_metadata_keys"`
	MetadataUpdateRateSeconds int `json:"metadata_update_rate_seconds"`
}

func LoadConfigFromDB(ctx context.Context, logger runtime.Logger, db *sql.DB) (*GameConfig, error) {
	configCacheMu.RLock()
	if configCache != nil && time.Since(lastCacheUpdate) < configCacheTTL {
		cached := configCache["game_settings"]
		configCacheMu.RUnlock()
		cached.ServerTime = time.Now().Unix()
		return &cached, nil
	}

	configCacheMu.RUnlock()

	query := `
		SELECT config_value 
		FROM game_config 
		WHERE config_key = $1 AND is_active = true
	`

	row := db.QueryRowContext(ctx, query, "game_settings")

	var rawJSON []byte
	if err := row.Scan(&rawJSON); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("game configuration not found in database")
		}
		return nil, fmt.Errorf("database error loading config: %w", err)
	}

	var config GameConfig
	if err := json.Unmarshal(rawJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	configCacheMu.Lock()
	if configCache == nil {
		configCache = make(map[string]GameConfig)
	}
	configCache["game_settings"] = config
	lastCacheUpdate = time.Now()
	configCacheMu.Unlock()

	config.ServerTime = time.Now().Unix()
	logger.Info("Game config loaded from PostgreSQL and cached")
	return &config, nil
}

func UpdateConfigInDB(ctx context.Context, logger runtime.Logger, db *sql.DB, key string, val map[string]interface{}) error {
	jsonValue, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("Failed to marshal config: %w", err)
	}

	query := `
		INSERT INTO game_config (config_key, config_value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (config_key) 
		DO UPDATE SET config_value = $2, updated_at = NOW()
	`

	_, err = db.ExecContext(ctx, query, key, jsonValue)
	if err != nil {
		return fmt.Errorf("Failed to update config in database: %w", err)
	}

	configCacheMu.Lock()
	configCache = nil
	lastCacheUpdate = time.Time{}
	configCacheMu.Unlock()

	logger.Info("Game config updated in PostgreSQL, cache invalidated")
	return nil
}

func GetLimits(ctx context.Context, logger runtime.Logger, db *sql.DB) (*LimitsConfig, error) {
	config, err := LoadConfigFromDB(ctx, logger, db)
	if err != nil {
		return nil, err
	}

	return &config.Limits, nil
}
