# Notice

Currently project is largely untested and thus I expect many obvious misses and errors that were not caught during coding.

---

# Basic overview of how to run

## Prerequisites
* docker
* docker compose plugin
* golang 1.25.0 +

---

## Building and running

* Go to EstotyTest root directory and run

```bash
docker compose up
```

 This should build and run the entire project, ***do note*** that this may require privelege escalation in case docker is not properly set up.

---

## Testing functionality

The following bash commands should result in the appropriate end points being called

* Update metadata
```bash
curl -X POST "http://localhost:7350/v2/rpc/update_user_metadata" \
  -H "Authorization: Bearer <SESSION_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"metadata": {"player_level": 5, "favorite_weapon": "sword"}}'
```

* Get game config
```bash
curl -X POST "http://localhost:7350/v2/rpc/get_game_config" \
  -H "Authorization: Bearer <SESSION_TOKEN>" \
  -d '{}'
```

* Private S2S health check
```bash
curl -X POST "http://localhost:7350/v2/rpc/private_health_check?http_key=defaulthttpkey&unwrap" \
  -d '{}'
```

* Private S2S config update
```bash
curl -X POST "http://localhost:7350/v2/rpc/private_update_config?http_key=defaulthttpkey&unwrap" \
  -H "Content-Type: application/json" \
  -d '{"config_key": "game_settings", "config_value": {"version": "1.1.0", "announcement": "New update available!"}}'
```

* Direct PostgreSQL config query
```bash
docker exec -it postgres psql -U postgres -d nakama -c "SELECT config_key, config_value->>'version' as version FROM game_config;
```

