--- create the schema
CREATE TABLE IF NOT EXISTS game_config (
	id SEIRAL PRIMARY KEY,
	config_key VARCHAR(128) NOT NULL UNIQUE,
	config_value JSONB NOT NULL,
	description TEXT,
	is_active BOOLEAN DEFAULT true,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
	updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- fast look up indexes
CREATE INDEX IF NOT EXISTS idx_game_config_key ON game_config(config_key)
CREATE INDEX IF NOT EXISTS idx_game_config_active ON game_config(is_active)

-- default config
INSERT INTO game_config (config_key, config_value, description) VALUES
(
'game_settings', '{
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
}'::jsonb, 'Core game configuration settings'
)
ON CONFLICT (config_key) DO NOTHING;

-- migration tracking table
CREATE TABLE IF NOT EXISTS schema_migrations (
	version VARCHAR(255) PRIMARY KEY,
	applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

INSERT INTO schema_migrations(version) VALUES ('create_game_config')
ON CONFLICT (version) DO NOTHING;