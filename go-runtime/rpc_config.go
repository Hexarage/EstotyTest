package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

func GetGameConfig(ctx context.Context, logger runtime.Logger, db *sql.DB, nakM runtime.NakamaModule, payload string) (string, error) {
	userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
	if !ok || userId == "" {
		return "", errors.New("authentication required")
	}

	config, err := LoadConfigFromDB(ctx, logger, db)
	if err != nil {
		logger.Error("Failed to load game config from database: %v", err)
		return "", runtime.NewError("Failed to load game configuration", 500)
	}

	config.ServerTime = time.Now().Unix()

	//serialize to json
	configBytes, err := json.Marshal(config)
	if err != nil {
		logger.Error("Failed to marshal the config: %v", err)
		return "", runtime.NewError("Internal server error", 500)
	}

	logger.Debug("Game config retreived for user %s (version: %s)", userId, config.Version)
	return string(configBytes), nil
}
