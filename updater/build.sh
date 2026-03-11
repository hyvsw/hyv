#!/bin/bash

GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.ControlServerHost=$HYV_CONTROL_SERVER_HOST -X main.ControlServerPort=$HYV_CONTROL_SERVER_PORT"
mv -f hyv_updater ../server/static/downloads/darwin/arm64/hyv_updater

GOOS=windows GOARCH=amd64 go build -ldflags "-X main.ControlServerHost=$HYV_CONTROL_SERVER_HOST -X main.ControlServerPort=$HYV_CONTROL_SERVER_PORT"
mv -f hyv_updater.exe ../server/static/downloads/windows/amd64/hyv_updater.exe

GOOS=linux GOARCH=amd64 go build -ldflags "-X main.ControlServerHost=$HYV_CONTROL_SERVER_HOST -X main.ControlServerPort=$HYV_CONTROL_SERVER_PORT"
mv -f hyv_updater ../server/static/downloads/linux/amd64/hyv_updater
