package main

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	MaxMetadataKeyLength   = 64
	MaxMetadataValueLength = 4096
	MaxMetadataDepth       = 3
)

func ValidateMetadata(metadata map[string]interface{}, maxSizeBytes, maxKeys int) error {
	if metadata == nil {
		return errors.New("Metadata cannot be nil")
	}

	if len(metadata) > maxKeys {
		return fmt.Errorf("Metadata exceeds maximum key count: %d > %d", len(metadata), maxKeys)
	}

	size := 0
	for key, val := range metadata {
		if len(key) > MaxMetadataKeyLength {
			return fmt.Errorf("Metadata key too long: %s (max %d characters)", key, MaxMetadataKeyLength)
		}

		for _, r := range key { // check format, obviously can be improved
			if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r != '_' && r != '.' {
				return fmt.Errorf("invalid character in metadata key: %s", key)
			}
		}

		valueSize, err := validateValue(val, 1)
		if err != nil {
			return fmt.Errorf("Invalid value for key %s: %w", key, err)
		}

		size += len(key) + valueSize

		if size > maxSizeBytes {
			return fmt.Errorf("Metadata exceeds maximum size %d > %d bytes", size, maxSizeBytes)
		}
	}

	return nil
}

func validateValue(value interface{}, depth int) (int, error) {
	if depth > MaxMetadataDepth {
		return 0, errors.New("Metadata nesting too deep")
	}

	switch v := value.(type) {
	case string:
		if len(v) > MaxMetadataValueLength {
			return 0, fmt.Errorf("String value too long: %d > %d", len(v), MaxMetadataValueLength)
		}

		return len(v), nil
	case float64, int, int64, bool:
		jsonBytes, _ := json.Marshal(v)
		return len(jsonBytes), nil
	case map[string]interface{}:
		size := 0
		for k, val := range v {
			if len(k) > MaxMetadataKeyLength {
				return 0, fmt.Errorf("Nested key too long: %s", k)
			}

			valSize, err := validateValue(val, depth+1)
			if err != nil {
				return 0, err
			}

			size += len(k) + valSize
		}
		return size, nil
	case []interface{}:
		size := 0
		for _, item := range v {
			itemSize, err := validateValue(item, depth+1)
			if err != nil {
				return 0, err
			}

			size += itemSize
		}
		return size, nil
	case nil:
		return 4, nil
	default:
		return 0, fmt.Errorf("Unsupported metadata value of type: %T", v)
	}
}
