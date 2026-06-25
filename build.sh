#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$ROOT_DIR/bin}"
TARGET="${TARGET:-}"

info()  { echo -e "\033[36m==>\033[0m $*"; }
error() { echo -e "\033[31mERROR:\033[0m $*"; exit 1; }

# --- Build frontend ---
info "Building frontend..."
pushd "$ROOT_DIR/web" > /dev/null
[ -d "node_modules" ] || npm ci
npm run build
popd > /dev/null

# --- Copy dist to embed location ---
info "Copying frontend dist to embed location..."
rm -rf "$ROOT_DIR/cmd/server/dist"
cp -r "$ROOT_DIR/web/dist" "$ROOT_DIR/cmd/server/dist"

# --- Build Go binary ---
mkdir -p "$OUTPUT_DIR"

LDFLAGS="-s -w"
OUTPUT_NAME="server"

if [ -n "$TARGET" ]; then
  GOOS="${TARGET%%/*}"
  GOARCH="${TARGET##*/}"
  export GOOS GOARCH
  info "Building Go binary for ${GOOS}/${GOARCH}..."
  if [ "$GOOS" = "windows" ]; then
    OUTPUT_NAME="server.exe"
  fi
else
  info "Building Go binary for host platform..."
  case "$(uname -s)" in
    MINGW*|MSYS*|CYGWIN*) OUTPUT_NAME="server.exe" ;;
  esac
fi

CGO_ENABLED=0 go build -o "$OUTPUT_DIR/$OUTPUT_NAME" -ldflags="$LDFLAGS" "$ROOT_DIR/cmd/server"

# --- Restore embed placeholder ---
rm -rf "$ROOT_DIR/cmd/server/dist"
mkdir -p "$ROOT_DIR/cmd/server/dist"
cat > "$ROOT_DIR/cmd/server/dist/index.html" << 'EOF'
<!doctype html><html lang="zh-CN"><head><meta charset="UTF-8"/><title>SimpleHub</title></head><body><div id="root">Placeholder</div></body></html>
EOF

FILE_SIZE="$(du -h "$OUTPUT_DIR/$OUTPUT_NAME" | cut -f1)"
echo ""
echo -e "\033[32mDone! Binary: $OUTPUT_DIR/$OUTPUT_NAME ($FILE_SIZE)\033[0m"
echo -e "\033[32mRun: $OUTPUT_DIR/$OUTPUT_NAME\033[0m"
