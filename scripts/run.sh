#!/usr/bin/env bash

set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH='' cd -- "${script_dir}/.." && pwd)
deps_dockerfile="${repo_dir}/deploy/docker/Dockerfile.battle-deps"
deps_hash=$(sha256sum "${deps_dockerfile}" | awk '{print $1}')
deps_image="game-demo-battle-deps:input-${deps_hash}"

cd "${repo_dir}"

start_stack() {
	if ! docker image inspect "${deps_image}" >/dev/null 2>&1; then
		BATTLE_DEPS_IMAGE="${deps_image}" \
			docker compose --profile build-deps build battle-deps
	fi

	BATTLE_DEPS_IMAGE="${deps_image}" docker compose up -d --build
}

is_stack_running() {
	[[ -n "$(docker compose ps --status running --quiet)" ]]
}

case "${1-}" in
	"")
		docker compose down --remove-orphans
		start_stack
		;;
	start)
		if is_stack_running; then
			echo "A service instance is already running. Stop it before starting again."
			exit 0
		fi
		start_stack
		;;
	stop)
		docker compose down --remove-orphans
		;;
	*)
		echo "Usage: ${0##*/} [start|stop]" >&2
		exit 2
		;;
esac
