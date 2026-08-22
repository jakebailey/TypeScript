#!/bin/bash

set -euo pipefail
cd "$(dirname "$0")"

out_dir="$PWD/built/tinygo"
tmp_dir="$out_dir/tmp"
workload="$PWD/tsc/testdata/tests/cases/compiler/2dArrays.ts"
go_root="$(GOTOOLCHAIN=go1.26.0 go env GOROOT)"

mkdir -p "$tmp_dir"

[[ -f "$out_dir/tsc" ]] || "$go_root/bin/go" -C tsc build -o "$out_dir/tsc" ./cmd/tsc
[[ -f "$out_dir/tsc-go.wasm" ]] || GOOS=wasip1 GOARCH=wasm "$go_root/bin/go" -C tsc build -o "$out_dir/tsc-go.wasm" ./cmd/tsc
[[ -f "$out_dir/tsc-tinygo.wasm" ]] || (
    cd tsc
    PATH="$go_root/bin:$PATH" GOROOT="$go_root" TMPDIR="$tmp_dir" tinygo build -target=wasip1 -o "$out_dir/tsc-tinygo.wasm" ./cmd/tsc
)

hyperfine -w 1 \
    -n 'native' \
    "'$out_dir/tsc' --ignoreConfig --noLib --noEmit --singleThreaded --listFilesOnly '$workload'" \
    -n 'go wasm' \
    "wasmtime run --dir '$PWD' --env HOME='$HOME' --env PATH='$PATH' -W max-wasm-stack=1048576 '$out_dir/tsc-go.wasm' --ignoreConfig --noLib --noEmit --singleThreaded --listFilesOnly '$workload'" \
    -n 'tinygo wasm' \
    "wasmtime run --dir '$PWD' --env HOME='$HOME' --env PATH='$PATH' -W max-wasm-stack=1048576 '$out_dir/tsc-tinygo.wasm' --ignoreConfig --noLib --noEmit --singleThreaded --listFilesOnly '$workload'"
