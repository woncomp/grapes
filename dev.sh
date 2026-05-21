#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$SCRIPT_DIR"

MODE=""

set_mode() {
    if [ -z "$MODE" ]; then
        MODE=$1
        return 0
    fi

    if [ "$MODE" = "$1" ]; then
        return 0
    fi

    echo "error: cannot combine build and release modes" >&2
    exit 1
}

EXTRA_ARG_COUNT=0

for arg in "$@"; do
    case "$arg" in
        -b|--build)
            set_mode build
            ;;
        -r|--release)
            set_mode release
            ;;
        *)
            EXTRA_ARG_COUNT=$((EXTRA_ARG_COUNT + 1))
            ;;
    esac
done

case "$MODE" in
    build)
        if [ "$EXTRA_ARG_COUNT" -ne 0 ]; then
            echo "error: build mode does not accept extra arguments" >&2
            exit 1
        fi
        mkdir -p ./bin
        go build -o ./bin/grapes ./cmd/grapes
        ;;
    release)
        if [ "$EXTRA_ARG_COUNT" -ne 0 ]; then
            echo "error: release mode does not accept extra arguments" >&2
            exit 1
        fi
        goreleaser release --snapshot --clean
        ;;
    *)
        go run ./cmd/grapes "$@"
        ;;
esac
