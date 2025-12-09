package main

import "github.com/kardianos/service"

func logLocation() string {
	return "C:\\ProgramData\\hyv\\hyv_updater.log"
}

func getPlatformAgentConfig() *service.Config {
	return &service.Config{
		Name:             "hyvagent",
		DisplayName:      "hyv agent",
		Description:      "Hyv Agent",
		Executable:       "C:\\ProgramData\\hyv\\hyv_agent.exe",
		WorkingDirectory: "C:\\Windows\\System32",
	}
}

func getPlatformUpdaterConfig() *service.Config {
	return &service.Config{
		Name:             "hyv_updater",
		DisplayName:      "hyv Updater",
		Description:      "Hyv Updater",
		Executable:       "C:\\ProgramData\\hyv\\hyv_updater.exe",
		WorkingDirectory: "C:\\Windows\\System32",
	}
}
