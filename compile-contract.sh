#!/usr/bin/env bash

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

protoc \
  --go_out "${SCRIPT_DIR}" --go_opt paths=source_relative \
  --go-grpc_out "${SCRIPT_DIR}" --go-grpc_opt paths=source_relative \
  --proto_path "${SCRIPT_DIR}" \
  contract/messenger.proto
