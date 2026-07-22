#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

go_protos=(
    proto/state/v1/state.proto
    proto/rcenter/v1/rcenter.proto
    proto/battle/v1/battle.proto
)

battle_cpp_out="battle-server/generated"
mkdir -p "$battle_cpp_out"

protoc \
    --go_out=. --go_opt=module=server \
    --go-grpc_out=. --go-grpc_opt=module=server \
    "${go_protos[@]}"

protoc \
    -I . \
    --cpp_out="$battle_cpp_out" \
    --grpc_out="$battle_cpp_out" \
    --plugin=protoc-gen-grpc="$(command -v grpc_cpp_plugin)" \
    proto/battle/v1/battle.proto
