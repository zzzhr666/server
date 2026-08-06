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
BATTLE_BUILD_DIR="${BATTLE_BUILD_DIR:-battle-server/cmake-build-release-wsl}"
BATTLE_SERVER_BIN="${BATTLE_SERVER_BIN:-${BATTLE_BUILD_DIR}/battle_server}"
BATTLE_1_NODE_NAME="${BATTLE_1_NODE_NAME:-battle-1}"
BATTLE_1_CONTROL_PORT="${BATTLE_1_CONTROL_PORT:-9101}"
BATTLE_1_UDP_PORT="${BATTLE_1_UDP_PORT:-7001}"
BATTLE_1_MAX_PLAYERS="${BATTLE_1_MAX_PLAYERS:-100}"
BATTLE_2_NODE_NAME="${BATTLE_2_NODE_NAME:-battle-2}"
BATTLE_2_CONTROL_PORT="${BATTLE_2_CONTROL_PORT:-9102}"
BATTLE_2_UDP_PORT="${BATTLE_2_UDP_PORT:-7002}"
BATTLE_2_MAX_PLAYERS="${BATTLE_2_MAX_PLAYERS:-100}"
BATTLE_UDP_PUBLIC_HOST="${BATTLE_UDP_PUBLIC_HOST:-127.0.0.1}"
LOGIC_1_PORT="${LOGIC_1_PORT:-8081}"
LOGIC_2_PORT="${LOGIC_2_PORT:-8082}"
START_NGINX="${START_NGINX:-1}"
NGINX_PREFIX="${ROOT_DIR}/tmp/nginx"
NGINX_CONF="${ROOT_DIR}/deploy/nginx/logic.conf"
LOG_DIR="${LOG_DIR:-${ROOT_DIR}/tmp/logs}"

state_pid=""
rcenter_pid=""
battle_1_pid=""
battle_2_pid=""
logic_1_pid=""
logic_2_pid=""
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

ensure_tcp_port_available() {
	local name="$1"
	local port="$2"
	if port_in_use "$port"; then
		echo "${name} port ${port} is already in use; run '$0 stop' first." >&2
		return 1
	fi
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

udp_port_in_use() {
	local port="$1"
	if command -v ss >/dev/null 2>&1; then
		ss -lun "sport = :${port}" | grep -q ":${port}"
		return
	fi
	if command -v lsof >/dev/null 2>&1; then
		lsof -iUDP:"${port}" >/dev/null 2>&1
		return
	fi
	return 1
}

ensure_udp_port_available() {
	local name="$1"
	local port="$2"
	if udp_port_in_use "$port"; then
		echo "${name} UDP port ${port} is already in use; run '$0 stop' first." >&2
		return 1
	fi
}

stop_udp_listener() {
	local name="$1"
	local port="$2"
	local pids

	if ! command -v lsof >/dev/null 2>&1; then
		if udp_port_in_use "$port"; then
			echo "${name} UDP port ${port} is in use, but lsof is unavailable; stop it manually and retry." >&2
			return 1
		fi
		return 0
	fi

	pids="$(lsof -tiUDP:"${port}" 2>/dev/null || true)"
	if [[ -z "$pids" ]]; then
		return 0
	fi

	echo "Stopping existing ${name} UDP listener on :${port}..."
	kill $pids 2>/dev/null || true
	sleep 0.3
	pids="$(lsof -tiUDP:"${port}" 2>/dev/null || true)"
	if [[ -n "$pids" ]]; then
		echo "Force stopping existing ${name} UDP listener on :${port}..."
		kill -9 $pids 2>/dev/null || true
		sleep 0.1
	fi
	if udp_port_in_use "$port"; then
		echo "${name} UDP port ${port} is still in use after cleanup." >&2
		return 1
	fi
}

start_battle_server() {
	local node_name="$1"
	local control_port="$2"
	local udp_port="$3"
	local max_players="$4"
	local pid_name="$5"
	local pid

	echo "Starting battle-server ${node_name} (control :${control_port}, UDP :${udp_port})..."
	start_background "$node_name" "$pid_name" "$BATTLE_SERVER_BIN" \
		--node-name "$node_name" \
		--control-addr "127.0.0.1:${control_port}" \
		--udp-bind-addr "0.0.0.0:${udp_port}" \
		--udp-addr "${BATTLE_UDP_PUBLIC_HOST}:${udp_port}" \
		--rcenter-addr "${RCENTER_GRPC_HOST}:${RCENTER_GRPC_PORT}" \
		--max-players "$max_players"
	pid="${!pid_name}"
	sleep 0.5
	if ! kill -0 "$pid" 2>/dev/null; then
		echo "battle-server ${node_name} exited during startup" >&2
		return 1
	fi
}

start_background() {
	local log_name="$1"
	local pid_name="$2"
	shift 2

	nohup "$@" >"${LOG_DIR}/${log_name}.log" 2>&1 &
	printf -v "$pid_name" '%s' "$!"
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

stop_services() {
	sudo nginx -p "${NGINX_PREFIX}" -c "${NGINX_CONF}" -s stop >/dev/null 2>&1 || true
	stop_port_listener "nginx" "8080"
	stop_port_listener "logic-1" "$LOGIC_1_PORT"
	stop_port_listener "logic-2" "$LOGIC_2_PORT"
	stop_port_listener "battle-server ${BATTLE_1_NODE_NAME} control" "$BATTLE_1_CONTROL_PORT"
	stop_port_listener "battle-server ${BATTLE_2_NODE_NAME} control" "$BATTLE_2_CONTROL_PORT"
	stop_udp_listener "battle-server ${BATTLE_1_NODE_NAME}" "$BATTLE_1_UDP_PORT"
	stop_udp_listener "battle-server ${BATTLE_2_NODE_NAME}" "$BATTLE_2_UDP_PORT"
	stop_port_listener "rcenter-server" "$RCENTER_GRPC_PORT"
	stop_port_listener "state-server" "$STATE_GRPC_PORT"
	echo "Services stopped."
}

start_services() {
	ensure_tcp_port_available "state-server" "$STATE_GRPC_PORT"
	ensure_tcp_port_available "rcenter-server" "$RCENTER_GRPC_PORT"
	ensure_tcp_port_available "logic-1" "$LOGIC_1_PORT"
	ensure_tcp_port_available "logic-2" "$LOGIC_2_PORT"
	if [[ "$START_NGINX" == "1" ]]; then
		ensure_tcp_port_available "nginx" "8080"
	fi
	if [[ "$START_BATTLE_SERVER" == "1" ]]; then
		ensure_tcp_port_available "battle-server ${BATTLE_1_NODE_NAME} control" "$BATTLE_1_CONTROL_PORT"
		ensure_tcp_port_available "battle-server ${BATTLE_2_NODE_NAME} control" "$BATTLE_2_CONTROL_PORT"
		ensure_udp_port_available "battle-server ${BATTLE_1_NODE_NAME}" "$BATTLE_1_UDP_PORT"
		ensure_udp_port_available "battle-server ${BATTLE_2_NODE_NAME}" "$BATTLE_2_UDP_PORT"
	fi

	mkdir -p "$LOG_DIR"
	echo "Starting state-server..."
	start_background "state-server" state_pid go run ./cmd/state-server
	wait_for_state_grpc

	echo "Starting rcenter-server..."
	start_background "rcenter-server" rcenter_pid go run ./cmd/rcenter-server
	wait_for_rcenter_grpc

	if [[ "$START_BATTLE_SERVER" == "1" ]]; then
		if [[ "$BUILD_BATTLE_SERVER" == "1" ]]; then
			echo "Building battle-server..."
			cmake --build "$BATTLE_BUILD_DIR"
		fi
		if [[ ! -x "$BATTLE_SERVER_BIN" ]]; then
			echo "battle-server binary not found or not executable: ${BATTLE_SERVER_BIN}" >&2
			return 1
		fi
		start_battle_server "$BATTLE_1_NODE_NAME" "$BATTLE_1_CONTROL_PORT" "$BATTLE_1_UDP_PORT" "$BATTLE_1_MAX_PLAYERS" battle_1_pid
		start_battle_server "$BATTLE_2_NODE_NAME" "$BATTLE_2_CONTROL_PORT" "$BATTLE_2_UDP_PORT" "$BATTLE_2_MAX_PLAYERS" battle_2_pid
	fi

	echo "Starting logic-server logic-1 on :${LOGIC_1_PORT}..."
	start_background "logic-1" logic_1_pid go run ./cmd/logic-server -p "${LOGIC_1_PORT}" --name logic-1
	echo "Starting logic-server logic-2 on :${LOGIC_2_PORT}..."
	start_background "logic-2" logic_2_pid go run ./cmd/logic-server -p "${LOGIC_2_PORT}" --name logic-2

	if [[ "$START_NGINX" == "1" ]]; then
		echo "Starting nginx reverse proxy on :8080..."
		mkdir -p "${NGINX_PREFIX}/logs" "${NGINX_PREFIX}/client_body_temp" "${NGINX_PREFIX}/proxy_temp"
		sudo nginx -p "${NGINX_PREFIX}" -c "${NGINX_CONF}"
	fi
	echo "Services started. Logs: ${LOG_DIR}"
}

usage() {
	echo "Usage: $0 [start|stop]" >&2
}

if (( $# > 1 )); then
	usage
	exit 1
fi

cd "$ROOT_DIR"
case "${1:-restart}" in
	start)
		start_services
		;;
	stop)
		stop_services
		;;
	restart)
		stop_services
		start_services
		;;
	*)
		usage
		exit 1
		;;
esac
