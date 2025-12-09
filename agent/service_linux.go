package main

import "github.com/kardianos/service"

func logLocation() string {
	return "/var/log/hyv/hyv_agent.log"
}

func getPlatformAgentConfig() *service.Config {
	return &service.Config{
		Name:        "hyv_agent",
		DisplayName: "hyv agent",
		Description: "Hyv Agent",
		Executable:  "/usr/bin/local/hyv/hyv_agent",
	}
}

func getPlatformUpdaterConfig() *service.Config {
	return &service.Config{
		Name:        "hyv_agent",
		DisplayName: "hyv agent",
		Description: "Hyv Agent",
		Executable:  "/usr/bin/local/hyv/hyv_updater",
	}
}
