#!/usr/bin/env bash

set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "${script_dir}/.." && pwd)
deps_dockerfile="${repo_dir}/deploy/docker/Dockerfile.battle-deps"
deps_hash=$(sha256sum "${deps_dockerfile}" | awk '{print $1}')
deps_image="game-demo-battle-deps:input-${deps_hash}"

cd "${repo_dir}"

if ! docker image inspect "${deps_image}" >/dev/null 2>&1; then
	BATTLE_DEPS_IMAGE="${deps_image}" \
		docker compose --profile build-deps build battle-deps
fi

BATTLE_DEPS_IMAGE="${deps_image}" docker compose up -d --build "$@"
