#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

protoc \
    --go_out=. --go_opt=module=server \
    --go-grpc_out=. --go-grpc_opt=module=server \
    proto/state/v1/state.proto
