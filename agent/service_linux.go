package main

import "github.com/kardianos/service"

func logLocation() string {
	return "/var/log/hyv/hyv_agent.log"
}

func getPlatformAgentConfig() *service.Config {
	return &service.Config{
		Name:        "hyv_agent",
		DisplayName: "HYV Agent",
		Description: "HYV Agent",
		Executable:  "/usr/bin/local/hyv/hyv_agent",
	}
}

func getPlatformUpdaterConfig() *service.Config {
	return &service.Config{
		Name:        "hyv_agent",
		DisplayName: "HYV Agent",
		Description: "HYV Agent",
		Executable:  "/usr/bin/local/hyv/hyv_updater",
	}
}
