#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_RPC_HOST="${STATE_RPC_HOST:-127.0.0.1}"
STATE_RPC_PORT="${STATE_RPC_PORT:-9001}"
STATE_RPC_TIMEOUT_SECONDS="${STATE_RPC_TIMEOUT_SECONDS:-10}"

state_pid=""
logic_pid=""

cleanup() {
	if [[ -n "$logic_pid" ]] && kill -0 "$logic_pid" 2>/dev/null; then
		kill "$logic_pid" 2>/dev/null || true
	fi
	if [[ -n "$state_pid" ]] && kill -0 "$state_pid" 2>/dev/null; then
		kill "$state_pid" 2>/dev/null || true
	fi
	wait 2>/dev/null || true
}

wait_for_state_rpc() {
	local deadline=$((SECONDS + STATE_RPC_TIMEOUT_SECONDS))
	while ((SECONDS < deadline)); do
		if (echo >"/dev/tcp/${STATE_RPC_HOST}/${STATE_RPC_PORT}") >/dev/null 2>&1; then
			return 0
		fi
		if ! kill -0 "$state_pid" 2>/dev/null; then
			echo "state-server exited before RPC port became ready" >&2
			return 1
		fi
		sleep 0.2
	done
	echo "state-server RPC port ${STATE_RPC_HOST}:${STATE_RPC_PORT} was not ready within ${STATE_RPC_TIMEOUT_SECONDS}s" >&2
	return 1
}

trap cleanup EXIT INT TERM

cd "$ROOT_DIR"

echo "Starting state-server..."
go run ./cmd/state-server &
state_pid=$!

wait_for_state_rpc

echo "Starting logic-server..."
go run ./cmd/logic-server &
logic_pid=$!

wait "$logic_pid"
