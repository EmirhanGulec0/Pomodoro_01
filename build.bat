@echo off
rem WorkX.exe derlemesi (Windows). Gereken: Go 1.24+ (https://go.dev/dl/)
go build -trimpath -ldflags "-H=windowsgui -s -w" -o WorkX.exe .
if errorlevel 1 exit /b 1
echo WorkX.exe hazir.
