package main

import "github.com/kardianos/service"

func logLocation() string {
	return "C:\\ProgramData\\hyv\\hyv_agent.log"
}

func getPlatformAgentConfig() *service.Config {
	return &service.Config{
		Name:             "hyv_agent",
		DisplayName:      "HYV agent",
		Description:      "HYV Agent",
		Executable:       "C:\\ProgramData\\hyv\\hyv_agent.exe",
		WorkingDirectory: "C:\\Windows\\System32",
	}
}

func getPlatformUpdaterConfig() *service.Config {
	return &service.Config{
		Name:             "hyv_updater",
		DisplayName:      "HYV updater",
		Description:      "HYV Updater",
		Executable:       "C:\\ProgramData\\hyv\\hyv_updater.exe",
		WorkingDirectory: "C:\\Windows\\System32",
	}
}
