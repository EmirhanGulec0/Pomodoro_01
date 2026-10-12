#!/bin/sh
# WorkX.exe çapraz derlemesi (Linux/macOS → Windows x64). Gereken: Go 1.24+
set -e
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H=windowsgui -s -w" -o WorkX.exe .
echo "WorkX.exe hazır."
