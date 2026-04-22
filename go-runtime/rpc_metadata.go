package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/heroiclabs/nakama-common/runtime"
)

type UpdateUserMetadataRequest struct {
	Metadata map[string]interface{} `json:"metadata"`
}

type UpdateUserMetadataResponse struct {
	Success  bool                   `json:"success"`
	Message  string                 `json:"message"`
	Metadata map[string]interface{} `json:"metadata"`
}

func UpdateUserMetadata(ctx context.Context, logger runtime.Logger, db *sql.DB, nakM runtime.NakamaModule, payload string) (string, error) {
	logger.Info("UpdateUserMetadata RPC entered...")
	logger.Info("Getting context")
	userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
	if !ok || userId == "" {
		return "", runtime.NewError("No user Id in context", 400)
	}

	logger.Info("Parsing payload")
	var req UpdateUserMetadataRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		logger.Error("Failed parsing the payload: %v", err)
		return "", runtime.NewError("Invalid JSON payload", 400)
	}

	logger.Info("Validating metadata")
	if req.Metadata == nil || len(req.Metadata) == 0 {
		return "", runtime.NewError("Metadata field is required and cannot be empty", 400)
	}

	logger.Info("Fetching account")
	limits, err := GetLimits(ctx, logger, db)
	if err != nil {
		logger.Error("Failed to load limits from config: %v", err)

		limits = &LimitsConfig{
			MaxMetadataSizeBytes:      16384,
			MaxMetadataKeys:           50,
			MetadataUpdateRateSeconds: 5,
		}
	}

	if err := ValidateMetadata(req.Metadata, limits.MaxMetadataSizeBytes, limits.MaxMetadataKeys); err != nil {
		logger.Warn("Metadata validation failed for user %s: %v", userId, err)
		return "", runtime.NewError(fmt.Sprintf("Validation failed: %s", err.Error()), 400)
	}

	account, err := nakM.AccountGetId(ctx, userId)
	if err != nil {
		logger.Error("Failed to get account for user %s: %v", userId, err)
		return "", runtime.NewError("Failed to retrieve user account", 500)
	}

	logger.Info("Merging metadata")
	mergedData := make(map[string]interface{})

	if account.User.Metadata != "" { // TODO: Unfuck this
		// merge into mergedData
	}

	for k, v := range req.Metadata {
		mergedData[k] = v
	}

	if err := ValidateMetadata(mergedData, limits.MaxMetadataSizeBytes, limits.MaxMetadataKeys); err != nil {
		return "", runtime.NewError("Merged metadata is too large", 400)
	}

	logger.Info("Updating account metadata")
	// genuinely bad interface
	err = nakM.AccountUpdateId(ctx, userId, "", mergedData, "", "", "", "", "") // empty strings are for no change

	if err != nil {
		logger.Error("Failed to update user account metadata: %v", err)
		return "", runtime.NewError("Failed to update account metadata", 500)
	}

	logger.Info("Building response")
	response := UpdateUserMetadataResponse{
		Success:  true,
		Message:  "Metadata updated successfully",
		Metadata: mergedData,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		logger.Error("Failed to marshal response: %v", err)
		return "", runtime.NewError("Internal server error", 500)
	}

	logger.Info("User %s updated metadata successfully (%d keys)", userId, len(mergedData))
	return string(responseBytes), nil
}
