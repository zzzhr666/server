#!/usr/bin/env bash
set -euo pipefail

REDIS_PATTERN="${REDIS_KEY_PATTERN:-game:*}"
MONGO_DATABASE="${MONGO_DATABASE:-game}"

if [[ -z "$MONGO_DATABASE" ]]; then
	echo "MONGO_DATABASE must not be empty" >&2
	exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
	echo "docker command is required" >&2
	exit 1
fi

redis_keys="$(docker compose exec -T redis redis-cli --scan --pattern "$REDIS_PATTERN")"
if [[ -n "$redis_keys" ]]; then
	mapfile -t keys <<< "$redis_keys"
else
	keys=()
fi

if ((${#keys[@]} > 0)); then
	docker compose exec -T redis redis-cli DEL "${keys[@]}" >/dev/null
	echo "Deleted ${#keys[@]} Redis key(s) matched pattern: $REDIS_PATTERN"
else
	echo "No Redis keys matched pattern: $REDIS_PATTERN"
fi

docker compose exec -e "MONGO_DATABASE=$MONGO_DATABASE" -T mongo mongosh --quiet --eval \
	'db.getSiblingDB(process.env.MONGO_DATABASE).dropDatabase()' >/dev/null
echo "Dropped MongoDB database: $MONGO_DATABASE"
