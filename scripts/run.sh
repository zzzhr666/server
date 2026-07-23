#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_GRPC_HOST="${STATE_GRPC_HOST:-127.0.0.1}"
STATE_GRPC_PORT="${STATE_GRPC_PORT:-9001}"
STATE_GRPC_TIMEOUT_SECONDS="${STATE_GRPC_TIMEOUT_SECONDS:-10}"
RCENTER_GRPC_HOST="${RCENTER_GRPC_HOST:-127.0.0.1}"
RCENTER_GRPC_PORT="${RCENTER_GRPC_PORT:-9002}"
RCENTER_GRPC_TIMEOUT_SECONDS="${RCENTER_GRPC_TIMEOUT_SECONDS:-10}"
START_BATTLE_SERVER="${START_BATTLE_SERVER:-1}"
BUILD_BATTLE_SERVER="${BUILD_BATTLE_SERVER:-1}"
BATTLE_BUILD_DIR="${BATTLE_BUILD_DIR:-battle-server/cmake-build-debug-wsl}"
BATTLE_SERVER_BIN="${BATTLE_SERVER_BIN:-${BATTLE_BUILD_DIR}/battle_server}"
BATTLE_CONTROL_PORT="${BATTLE_CONTROL_PORT:-9101}"
LOGIC_1_PORT="${LOGIC_1_PORT:-8081}"
LOGIC_2_PORT="${LOGIC_2_PORT:-8082}"
START_NGINX="${START_NGINX:-1}"
NGINX_PREFIX="${ROOT_DIR}/tmp/nginx"
NGINX_CONF="${ROOT_DIR}/deploy/nginx/logic.conf"

state_pid=""
rcenter_pid=""
battle_pid=""
logic_1_pid=""
logic_2_pid=""
nginx_started=""

cleanup() {
	if [[ "$nginx_started" == "1" ]]; then
		sudo nginx -p "${NGINX_PREFIX}" -c "${NGINX_CONF}" -s stop >/dev/null 2>&1 || true
	fi
	if [[ -n "$logic_1_pid" ]] && kill -0 "$logic_1_pid" 2>/dev/null; then
		kill "$logic_1_pid" 2>/dev/null || true
	fi
	if [[ -n "$logic_2_pid" ]] && kill -0 "$logic_2_pid" 2>/dev/null; then
		kill "$logic_2_pid" 2>/dev/null || true
	fi
	if [[ -n "$rcenter_pid" ]] && kill -0 "$rcenter_pid" 2>/dev/null; then
		kill "$rcenter_pid" 2>/dev/null || true
	fi
	if [[ -n "$battle_pid" ]] && kill -0 "$battle_pid" 2>/dev/null; then
		kill "$battle_pid" 2>/dev/null || true
	fi
	if [[ -n "$state_pid" ]] && kill -0 "$state_pid" 2>/dev/null; then
		kill "$state_pid" 2>/dev/null || true
	fi
	wait 2>/dev/null || true
}

port_in_use() {
	local port="$1"
	if command -v ss >/dev/null 2>&1; then
		ss -ltn "sport = :${port}" | grep -q ":${port}"
		return
	fi
	if command -v lsof >/dev/null 2>&1; then
		lsof -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
		return
	fi
	(echo >"/dev/tcp/127.0.0.1/${port}") >/dev/null 2>&1
}

stop_port_listener() {
	local name="$1"
	local port="$2"
	local pids

	if ! command -v lsof >/dev/null 2>&1; then
		if port_in_use "$port"; then
			echo "${name} port ${port} is in use, but lsof is unavailable; stop it manually and retry." >&2
			return 1
		fi
		return 0
	fi

	pids="$(lsof -tiTCP:"${port}" -sTCP:LISTEN 2>/dev/null || true)"
	if [[ -z "$pids" ]]; then
		return 0
	fi

	echo "Stopping existing ${name} listener on :${port}..."
	kill $pids 2>/dev/null || true
	sleep 0.3

	pids="$(lsof -tiTCP:"${port}" -sTCP:LISTEN 2>/dev/null || true)"
	if [[ -n "$pids" ]]; then
		echo "Force stopping existing ${name} listener on :${port}..."
		kill -9 $pids 2>/dev/null || true
		sleep 0.1
	fi

	if port_in_use "$port"; then
		echo "${name} port ${port} is still in use after cleanup." >&2
		return 1
	fi
}

stop_existing_nginx() {
	if [[ "$START_NGINX" == "1" ]]; then
		sudo nginx -p "${NGINX_PREFIX}" -c "${NGINX_CONF}" -s stop >/dev/null 2>&1 || true
	fi
}

wait_for_state_grpc() {
	local deadline=$((SECONDS + STATE_GRPC_TIMEOUT_SECONDS))
	while ((SECONDS < deadline)); do
		if (echo >"/dev/tcp/${STATE_GRPC_HOST}/${STATE_GRPC_PORT}") >/dev/null 2>&1; then
			return 0
		fi
		if ! kill -0 "$state_pid" 2>/dev/null; then
			echo "state-server exited before gRPC port became ready" >&2
			return 1
		fi
		sleep 0.2
	done
	echo "state-server gRPC port ${STATE_GRPC_HOST}:${STATE_GRPC_PORT} was not ready within ${STATE_GRPC_TIMEOUT_SECONDS}s" >&2
	return 1
}

wait_for_rcenter_grpc() {
	local deadline=$((SECONDS + RCENTER_GRPC_TIMEOUT_SECONDS))
	while ((SECONDS < deadline)); do
		if (echo >"/dev/tcp/${RCENTER_GRPC_HOST}/${RCENTER_GRPC_PORT}") >/dev/null 2>&1; then
			return 0
		fi
		if ! kill -0 "$rcenter_pid" 2>/dev/null; then
			echo "rcenter-server exited before gRPC port became ready" >&2
			return 1
		fi
		sleep 0.2
	done
	echo "rcenter-server gRPC port ${RCENTER_GRPC_HOST}:${RCENTER_GRPC_PORT} was not ready within ${RCENTER_GRPC_TIMEOUT_SECONDS}s" >&2
	return 1
}

trap cleanup EXIT INT TERM

cd "$ROOT_DIR"

stop_existing_nginx
stop_port_listener "state-server" "$STATE_GRPC_PORT"
stop_port_listener "rcenter-server" "$RCENTER_GRPC_PORT"
if [[ "$START_BATTLE_SERVER" == "1" ]]; then
	stop_port_listener "battle-server control" "$BATTLE_CONTROL_PORT"
fi
stop_port_listener "logic-1" "$LOGIC_1_PORT"
stop_port_listener "logic-2" "$LOGIC_2_PORT"
if [[ "$START_NGINX" == "1" ]]; then
	stop_port_listener "nginx" "8080"
fi

echo "Starting state-server..."
go run ./cmd/state-server &
state_pid=$!

wait_for_state_grpc

echo "Starting rcenter-server..."
go run ./cmd/rcenter-server &
rcenter_pid=$!

wait_for_rcenter_grpc

if [[ "$START_BATTLE_SERVER" == "1" ]]; then
	if [[ "$BUILD_BATTLE_SERVER" == "1" ]]; then
		echo "Building battle-server..."
		cmake --build "$BATTLE_BUILD_DIR"
	fi

	if [[ ! -x "$BATTLE_SERVER_BIN" ]]; then
		echo "battle-server binary not found or not executable: ${BATTLE_SERVER_BIN}" >&2
		exit 1
	fi

	echo "Starting battle-server..."
	"$BATTLE_SERVER_BIN" &
	battle_pid=$!
	sleep 0.5
	if ! kill -0 "$battle_pid" 2>/dev/null; then
		echo "battle-server exited during startup" >&2
		exit 1
	fi
fi

echo "Starting logic-server logic-1 on :${LOGIC_1_PORT}..."
go run ./cmd/logic-server -p "${LOGIC_1_PORT}" --name logic-1 &
logic_1_pid=$!

echo "Starting logic-server logic-2 on :${LOGIC_2_PORT}..."
go run ./cmd/logic-server -p "${LOGIC_2_PORT}" --name logic-2 &
logic_2_pid=$!

if [[ "$START_NGINX" == "1" ]]; then
	echo "Starting nginx reverse proxy on :8080..."
	mkdir -p \
		"${NGINX_PREFIX}/logs" \
		"${NGINX_PREFIX}/client_body_temp" \
		"${NGINX_PREFIX}/proxy_temp"
	sudo nginx -p "${NGINX_PREFIX}" -c "${NGINX_CONF}"
	nginx_started="1"
fi

wait_pids=("$logic_1_pid" "$logic_2_pid")
if [[ -n "$battle_pid" ]]; then
	wait_pids+=("$battle_pid")
fi
wait "${wait_pids[@]}"
