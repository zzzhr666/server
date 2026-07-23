#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

go_protos=(
	proto/state/v1/state.proto
	proto/rcenter/v1/rcenter.proto
	proto/battle/v1/battle.proto
	proto/battle/v1/session.proto
)

battle_cpp_protos=(
	proto/battle/v1/battle.proto
	proto/battle/v1/session.proto
	proto/rcenter/v1/rcenter.proto
)

battle_cpp_grpc_protos=(
	proto/battle/v1/battle.proto
	proto/rcenter/v1/rcenter.proto
)

battle_cpp_out="battle-server/generated"

mkdir -p "$battle_cpp_out"

generate_go() {
	protoc \
		--go_out=. --go_opt=module=server \
		--go-grpc_out=. --go-grpc_opt=module=server \
		"$@"
}

generate_cpp() {
	protoc \
		-I . \
		--cpp_out="$battle_cpp_out" \
		"$@"
}

generate_cpp_grpc() {
	protoc \
		-I . \
		--grpc_out="$battle_cpp_out" \
		--plugin=protoc-gen-grpc="$(command -v grpc_cpp_plugin)" \
		"$@"
}

remove_stale_outputs() {
	rm -f \
		battle-server/generated/proto/battle/v1/session.grpc.pb.cc \
		battle-server/generated/proto/battle/v1/session.grpc.pb.h
}

remove_stale_outputs

protoc \
	--version >/dev/null

generate_go "${go_protos[@]}"
generate_cpp "${battle_cpp_protos[@]}"
generate_cpp_grpc "${battle_cpp_grpc_protos[@]}"
