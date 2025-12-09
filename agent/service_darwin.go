package main

import (
	"github.com/kardianos/service"
)

func logLocation() string {
	return "/Library/Application Support/hyv/hyv_agent.log"
}

func getPlatformAgentConfig() *service.Config {
	return &service.Config{
		Name:             "hyv_agent",
		DisplayName:      "HYV Agent",
		Description:      "HYV Agent",
		Executable:       "/Applications/hyv/hyv_agent",
		WorkingDirectory: "/Applications/hyv",
	}
}

func getPlatformUpdaterConfig() *service.Config {
	return &service.Config{
		Name:             "hyv_updater",
		DisplayName:      "HYV Updater",
		Description:      "HYV Updater",
		Executable:       "/Applications/hyv/hyv_updater",
		WorkingDirectory: "/Applications/hyv",
	}
}
