package main

import (
	"github.com/kardianos/service"
)

func logLocation() string {
	return "/Library/Application Support/hyv/hyv_updater.log"
}

func getPlatformAgentConfig() *service.Config {
	return &service.Config{
		Name:             "hyv_agent",
		DisplayName:      "hyv agent",
		Description:      "Hyv Agent",
		Executable:       "/Applications/hyv/hyv_agent",
		WorkingDirectory: "/Applications/hyv",
	}
}

func getPlatformUpdaterConfig() *service.Config {
	return &service.Config{
		Name:             "hyv_updater",
		DisplayName:      "hyv updater",
		Description:      "Hyv Updater",
		Executable:       "/Applications/hyv/hyv_updater",
		WorkingDirectory: "/Applications/hyv",
	}
}
