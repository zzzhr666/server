#!/usr/bin/env bash
set -euo pipefail

REDIS_CLI="${REDIS_CLI:-redis-cli}"
PATTERN="${REDIS_KEY_PATTERN:-game:*}"

mapfile -t keys < <("$REDIS_CLI" --scan --pattern "$PATTERN")

if ((${#keys[@]} == 0)); then
	echo "No Redis keys matched pattern: $PATTERN"
	exit 0
fi

"$REDIS_CLI" DEL "${keys[@]}" >/dev/null
echo "Deleted ${#keys[@]} Redis key(s) matched pattern: $PATTERN"
