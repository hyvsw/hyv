package main

import "github.com/kardianos/service"

func logLocation() string {
	return "C:\\ProgramData\\hyv\\hyv_updater.log"
}

func getPlatformAgentConfig() *service.Config {
	return &service.Config{
		Name:             "hyvagent",
		DisplayName:      "HYV Agent",
		Description:      "HYV Agent",
		Executable:       "C:\\ProgramData\\hyv\\hyv_agent.exe",
		WorkingDirectory: "C:\\Windows\\System32",
	}
}

func getPlatformUpdaterConfig() *service.Config {
	return &service.Config{
		Name:             "hyv_updater",
		DisplayName:      "HYV Updater",
		Description:      "HYV Updater",
		Executable:       "C:\\ProgramData\\hyv\\hyv_updater.exe",
		WorkingDirectory: "C:\\Windows\\System32",
	}
}
