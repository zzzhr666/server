#!/usr/bin/env bash
set -euo pipefail

PATTERN="${REDIS_KEY_PATTERN:-game:*}"

mapfile -t keys < <(docker compose exec -T redis redis-cli --scan --pattern "$PATTERN")

if ((${#keys[@]} == 0)); then
	echo "No Redis keys matched pattern: $PATTERN"
	exit 0
fi

docker compose exec -T redis redis-cli DEL "${keys[@]}" >/dev/null
echo "Deleted ${#keys[@]} Redis key(s) matched pattern: $PATTERN"
