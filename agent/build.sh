#!/bin/bash

GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.ControlServerHost=$HYV_CONTROL_SERVER_HOST -X main.ControlServerPort=$HYV_CONTROL_SERVER_PORT"
mv -f hyv_agent ../server/static/downloads/darwin/arm64/hyv_agent

GOOS=windows GOARCH=amd64 go build -ldflags "-X main.ControlServerHost=$HYV_CONTROL_SERVER_HOST -X main.ControlServerPort=$HYV_CONTROL_SERVER_PORT"
mv -f hyv_agent.exe ../server/static/downloads/windows/amd64/hyv_agent.exe

#GOOS=linux GOARCH=amd64 go build -ldflags "-X main.ControlServerHost=$HYV_CONTROL_SERVER_HOST -X main.ControlServerPort=$HYV_CONTROL_SERVER_PORT"
#mv -f hyv_agent ../server/static/downloads/linux/amd64/hyv_agent
