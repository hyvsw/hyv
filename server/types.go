package main

import "time"

type windowsSystemData struct {
	BIOS struct {
		SerialNumber      string
		SMBIOSBIOSVersion string
	}
	Computer struct {
		Manufacturer string
		Model        string
	}
	BaseBoard struct {
		Product      string
		Manufacturer string
		SerialNumber string
	}
	OS struct {
		Caption        string
		Version        string
		OSArchitecture string
	}
	Product struct {
		UUID string
	}
}

// the struct into which darwinPowerMetrics plist output is parsed
type darwinPowerMetrics struct {
	IsDelta           bool        `plist:"is_delta"`
	ElapsedNS         int64       `plist:"elapsed_ns"`
	HWModel           string      `plist:"hw_model"`
	KernVersion       string      `plist:"kern_version"`
	KernBootArgs      string      `plist:"kern_bootargs"`
	KernBootTime      int64       `plist:"kern_boottime"`
	Timestamp         time.Time   `plist:"timestamp"`
	ProcessCoalitions []Coalition `plist:"coalitions"`
	Battery           struct {
		PercentCharge int `plist:"percent_charge"`
	} `plist:"battery"`
	Network struct {
		OutPackets    int     `plist:"opackets"`
		OutPacketRate float64 `plist:"opacket_rate"`
		InPackets     int     `plist:"ipackets"`
		InpacketRate  float64 `plist:"ipacket_rate"`
		OutBytes      int     `plist:"obytes"`
		OutByteRate   float64 `plist:"obyte_rate"`
		InBytes       int     `plist:"ibyte"`
		InByteRate    float64 `plist:"ibyte_rate"`
	} `plist:"network"`
	Disk struct {
		ReadOpsDiff    int     `plist:"rops_diff"`
		ReadOpsPerS    float64 `plist:"rops_per_s"`
		WriteOpsDiff   int     `plist:"wops_diff"`
		WriteOpsPerS   float64 `plist:"wops_per_s"`
		ReadBytesDiff  int     `plist:"rbytes_diff"`
		ReadBytesPerS  float64 `plist:"rbytes_per_s"`
		WriteBytesDiff int     `plist:"wbytes_diff"`
		WriteBytesPerS float64 `plist:"wbytes_per_s"`
	} `plist:"disk"`
	// Skipping Interrupts
	Processor struct {
		Clusters []struct {
			Name             string      `plist:"name"`
			HWResIDCounters  bool        `plist:"hw_resid_counters"`
			FreqHz           float64     `plist:"freq_hz"`
			IdleNS           int         `plist:"idle_ns"`
			IdleRatio        float64     `plist:"idle_ratio"`
			DVFMStates       []DVFMState `plist:"dvfm_states"`
			OnlineRatio      float64     `plist:"online_ratio"`
			RequestedMhz     int         `plist:"requested_mhz"`
			RecommendedCores float64     `plist:"recommended_cores"`
			CPUs             []CPU       `plist:"cpus"`
		} `plist:"clusters"`
		CPUPowerZonesEngaged float64 `plist:"cpu_power_zones_engaged"`
		CPUEnergy            int     `plist:"cpu_energy"`
		CPUPower             float64 `plist:"cpu_power"`
		GPUEnergy            int     `plist:"gpu_energy"`
		GPUPower             float64 `plist:"gpu_power"`
		ANEEnergy            int     `plist:"ane_energy"`
		ANEPower             float64 `plist:"ane_power"`
	} `plist:"processor"`
	ThermalPressure string `plist:"thermal_pressure"`
	SFI             struct {
		WindowSizeUS int `plist:"window_size_us"`
		SFIClasses   struct {
			SFIClassDarwinBG                int `plist:"SFI_CLASS_DARWIN_BG"`
			SFIClassAppNap                  int `plist:"SFI_CLASS_APP_NAP"`
			SFIClassManagedFocal            int `plist:"SFI_CLASS_MANAGED_FOCAL"`
			SFIClassManagedNonFocal         int `plist:"SFI_CLASS_MANAGED_NONFOCAL"`
			SFIClassDefaultFocal            int `plist:"SFI_CLASS_DEFAULT_FOCAL"`
			SFIClassDefaultNonFocal         int `plist:"SFI_CLASS_DEFAULT_NONFOCAL"`
			SFIClassOptedOut                int `plist:"SFI_CLASS_OPTED_OUT"`
			SFIClassUtility                 int `plist:"SFI_CLASS_UTILITY"`
			SFIClassLegacyFocal             int `plist:"SFI_CLASS_LEGACY_FOCAL"`
			SFIClassLegacyNonFocal          int `plist:"SFI_CLASS_LEGACY_NONFOCAL"`
			SFIClassUserInitiatedFocal      int `plist:"SFI_CLASS_USER_INITIATED_FOCAL"`
			SFIClassUserInitiatedNonFocal   int `plist:"SFI_CLASS_USER_INITIATED_NONFOCAL"`
			SFIClassUserInteractiveFocal    int `plist:"SFI_CLASS_USER_INTERACTIVE_FOCAL"`
			SFIClassUserInteractiveNonFocal int `plist:"SFI_CLASS_USER_INTERACTIVE_NONFOCAL"`
			SFIClassMaintenance             int `plist:"SFI_CLASS_MAINTENANCE"`
		} `plist:"sfi_classes"`
	} `plist:"sfi"`
	GPU struct {
		FreqHz           float64     `plist:"freq_hz"`
		IdleNS           int         `plist:"idle_ns"`
		IdleRatio        float64     `plist:"idle_ratio"`
		DVFMStates       []DVFMState `plist:"dvfm_states"`
		SWRequestedState []struct {
			SWReqState string  `plist:"sw_req_state"`
			UsedNS     int     `plist:"used_ns"`
			UsedRatio  float64 `plist:"used_ratio"`
		} `plist:"sw_requested_state"`
	} `plist:"gpu"`
}

type DVFMState struct {
	Freq      int     `plist:"freq"`
	UsedNS    int     `plist:"used_ns"`
	UsedRatio float64 `plist:"used_ratio"`
}

type CPU struct {
	CPU        int         `plist:"cpu"`
	FreqHz     float64     `plist:"freq_hz"`
	IdleNS     int         `plist:"idle_ns"`
	IdleRatio  float64     `plist:"idle_ratio"`
	DownNS     int         `plist:"down_ns"`
	DownRatio  float64     `plist:"down_ratio"`
	DVFMStates []DVFMState `plist:"dvfm_states"`
}

type Coalition struct {
	ID                     int     `plist:"id"`
	Name                   string  `plist:"name"`
	CPUTimeNS              int     `plist:"cputime_ns"`
	CPUTimeMSPerS          float64 `plist:"cputime_ms_per_s"`
	CPUTimeSampleMSPerS    float64 `plist:"cputime_sample_ms_per_s"`
	IntrWakeups            int     `plist:"intr_wakeups"`
	IntrWakeupsPerS        float64 `plist:"intr_wakeups_per_s"`
	IdleWakeups            int     `plist:"idle_wakeups"`
	IdleWakeupsPerS        float64 `plist:"idle_wakeups_per_s"`
	DiskIOBytesRead        float64 `plist:"diskio_bytes_read"`
	DiskIOBytesReadPerS    float64 `plist:"diskio_bytes_read_per_s"`
	DiskIOBytesWritten     float64 `plist:"diskio_bytes_written"`
	DiskIOBytesWrittenPerS float64 `plist:"diskio_bytes_written_per_s"`
	EnergyImpact           float64 `plist:"energy_impact"`
	EnergyImpactPerS       float64 `plist:"energy_impact_per_s"`
	CPUInstructions        float64 `plist:"cpu_instructions"`
	CPUCycles              float64 `plist:"cpu_cycles"`
	PCPUInstructions       float64 `plist:"pcpu_instructions"`
	PCPUCycles             float64 `plist:"pcpu_cycles"`
	Tasks                  []Task  `plist:"tasks"`
}

type Task struct {
	PID                      int           `plist:"pid"`
	Name                     string        `plist:"name"`
	StartedAbsTimeNS         int           `plist:"started_abstime_ns"`
	IntervalNS               int           `plist:"interval_ns"`
	CPUTimeNS                int           `plist:"cputime_ns"`
	CPUTimeMSPerS            float64       `plist:"cputime_ms_per_s"`
	CPUTimeSampleMSPerS      float64       `plist:"cputime_sample_ms_per_s"`
	CPUTimeUserlandRatio     float64       `plist:"cputime_userland_ratio"`
	IntrWakeups              int           `plist:"intr_wakeups"`
	IntrWakeupsPerS          float64       `plist:"intr_wakeups_per_s"`
	IdleWakeups              int           `plist:"idle_wakeups"`
	IdleWakeupsPerS          float64       `plist:"idle_wakeups_per_s"`
	TimerWakeups             []TimerWakeup `plist:"timer_wakeups"`
	DiskIOBytesRead          float64       `plist:"diskio_bytes_read"`
	DiskIOBytesReadPerS      float64       `plist:"diskio_bytes_read_per_s"`
	DiskIOBytesWritten       float64       `plist:"diskio_bytes_written"`
	DiskIOBytesWrittenPerS   float64       `plist:"diskio_bytes_written_per_s"`
	PageIns                  int           `plist:"pageins"`
	PageInsPerS              float64       `plist:"pageins_per_s"`
	QOSDisabledNS            int           `plist:"qos_disabled_ns"`
	QOSDisabledMSPerS        float64       `plist:"qos_disabled_ms_per_s"`
	QOSMaintenanceNS         int           `plist:"qos_maintenance_ns"`
	QOSMaintenanceMSPerS     float64       `plist:"qos_maintenance_ms_per_s"`
	QOSBackgroundNS          int           `plist:"qos_background_ns"`
	QOSBackgroundMSPerS      float64       `plist:"qos_background_ms_per_s"`
	QOSUtilityNS             int           `plist:"qos_utility_ns"`
	QOSUtilityMSPerS         float64       `plist:"qos_utility_ms_per_s"`
	QOSDefaultNS             int           `plist:"qos_default_ns"`
	QOSDefaultMSPerS         float64       `plist:"qos_default_ms_per_s"`
	QOSUserInitiatedNS       int           `plist:"qos_user_initiated_ns"`
	QOSUserInitiatedMSPerS   float64       `plist:"qos_user_initiated_ms_per_s"`
	QOSUserInteractiveNS     int           `plist:"qos_user_interactive_ns"`
	QOSUserInteractiveMSPerS float64       `plist:"qos_user_interactive_ms_per_s"`
	SFINS                    int           `plist:"sfi_ns"`
	SFIMSPerS                float64       `plist:"sfi_ms_per_s"`
	QOS                      struct {
		ThroughputTier int `plist:"throughput_tier"`
		LatencyTier    int `plist:"latency_tier"`
	} `plist:"qos"`
	ResponsiblePID      int     `plist:"responsidble_pid"`
	ParentPID           int     `plist:"parent_pid"`
	PTimeNS             int     `plist:"ptime_ns"`
	PTimeMSPerS         float64 `plist:"ptime_ms_per_s"`
	PTimeRatio          float64 `plist:"ptime_ratio"`
	EPSwitches          int     `plist:"epswitches"`
	EPSwitchesPerS      float64 `plist:"epswitches_per_s"`
	PacketsReceived     int     `plist:"packets_received"`
	PacketsReceivedPerS float64 `plist:"packets_received_per_s"`
	PacketsSent         int     `plist:"packets_sent"`
	PacketsSentPerS     float64 `plist:"packets_sent_per_s"`
	BytesReceived       int     `plist:"bytes_received"`
	BytesReceivedPerS   float64 `plist:"bytes_received_per_s"`
	BytesSent           int     `plist:"bytes_sent"`
	BytesSentPerS       float64 `plist:"bytes_sent_per_s"`
	EnergyImpact        float64 `plist:"energy_impact"`
	EnergyImpactPerS    float64 `plist:"energy_impact_per_s"`
	CPUInstructions     float64 `plist:"cpu_instructions"`
	CPUCycles           float64 `plist:"cpu_cycles"`
	PCPUInstructions    float64 `plist:"pcpu_instructions"`
	PCPUCycles          float64 `plist:"pcpu_cycles"`
}

type TimerWakeup struct {
	IntervalNS  int     `plist:"interval_ns"`
	Wakeups     int     `plist:"wakeups"`
	WakeupsPerS float64 `plist:"wakeups_per_s"`
}
