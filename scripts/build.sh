#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
go run github.com/vivaui/viva/cmd/viva package --format app
