package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"

	"github.com/heroiclabs/nakama-common/runtime"
)

func PrivateHealthCheck(ctx context.Context, logger runtime.Logger, db *sql.DB, nakM runtime.NakamaModule, payload string) (string, error) {
	userId, hasUserId := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)

	// s2s calls are supposed to have no user value
	if hasUserId && userId != "" {
		logger.Warn("Blocked client trying to call private s2s rpc from user id: %s", userId)
		return "", runtime.NewError("Endpoint is restricted to server-to-server calls only", 403)
	}

	httpKey, hasHttpKey := "", false // TODO: get http key from runtime (probably .(string)?)
	if !hasHttpKey && httpKey == "" {
		logger.Warn("Blocked unauthorized call to private s2s rpc")
		return "", runtime.NewError("Authentication required: HTTP key missing", 401)
	}

	expectedKey := os.Getenv("NAKAMA_RUNTIME_HTTP_KEY")
	if expectedKey == "" {
		// dev fallback?
		expectedKey = "defaulthttpkey" // TODO: change to something sane so it doesn't leak into prod?
		logger.Warn("Using default http key - set NAKAMA_RUNTIME_HTTP_KEY env variable for prod")
	}

	if httpKey != expectedKey {
		logger.Warn("Invalid HTTP key provided for private s2s rpc")
		return "", errors.New("Invalid authentication")
	}

	logger.Info("Checking database connection")
	if err := db.PingContext(ctx); err != nil {
		logger.Error("Health check failed - database unreachable: %v", err)
		return "", runtime.NewError("Service unhealthy: database connection failed", 503)
	}

	logger.Info("S2S health check passed from remote server")

	return "", nil // returning an empty string should be HTTP 200 with empty body
}

type PrivateUpdateConfigRequest struct {
	ConfigKey   string                 `json:"config_key"`
	ConfigValue map[string]interface{} `json:"config_value"`
}

func PrivateUpdateConfig(ctx context.Context, logger runtime.Logger, db *sql.DB, nakM runtime.NakamaModule, payload string) (string, error) {
	userId, hasUserID := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
	if hasUserID && userId != "" {
		return "", runtime.NewError("S2S only", 403)
	}

	httpKey, hasHTTPKey := "", false // TODO: same as PrivateHealthCheck
	if !hasHTTPKey || httpKey == "" {
		return "", runtime.NewError("HTTP key required", 401)
	}

	expectedKey := os.Getenv("NAKAMA_RUNTIME_HTTP_KEY")
	if expectedKey == "" {
		expectedKey = "defaulthttpkey"
	}

	if httpKey != expectedKey {
		return "", runtime.NewError("Invalid credentials", 401)
	}

	var req PrivateUpdateConfigRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		return "", runtime.NewError("Invalid JSON payload", 400)
	}

	if req.ConfigKey == "" || req.ConfigValue == nil {
		return "", runtime.NewError("config_key and config_value required", 400)
	}

	// update in postgresql
	if err := UpdateConfigInDB(ctx, logger, db, req.ConfigKey, req.ConfigValue); err != nil {
		logger.Error("Failed to update config: %v", err)
		return "", runtime.NewError("Failed to update configuration", 500)
	}

	return `{"success": true, "message": "Configuration updated"}`, nil
}
