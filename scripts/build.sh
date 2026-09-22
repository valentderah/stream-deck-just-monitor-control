#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/dist/com.valentderah.just-monitor-control.sdPlugin"

rm -rf "$OUT"
mkdir -p "$OUT/bin"
export CGO_ENABLED=0
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o "$OUT/bin/plugin.exe" "$ROOT/cmd/plugin"

cp "$ROOT/assets/manifest.json" "$OUT/"
cp "$ROOT/assets/"*.json "$OUT/" 2>/dev/null || true
cp -R "$ROOT/assets/ui" "$OUT/ui"
cp -R "$ROOT/assets/imgs" "$OUT/imgs"

echo "Built $OUT"
