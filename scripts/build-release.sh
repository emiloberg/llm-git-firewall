#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-dev}"
OUTPUT_DIR="dist"

mkdir -p "$OUTPUT_DIR"

platforms=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
)

for platform in "${platforms[@]}"; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"
  output="$OUTPUT_DIR/llm-git-firewall-${GOOS}-${GOARCH}"

  echo "Building $GOOS/$GOARCH..."
  GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w" -o "$output" ./cmd/llm-git-firewall
done

echo "Done. Binaries in $OUTPUT_DIR/"
ls -lh "$OUTPUT_DIR/"
