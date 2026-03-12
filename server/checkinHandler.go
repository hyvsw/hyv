package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

type checkinData struct {
	ID       int
	HyvID    string
	Hostname string
	OS       string
	Serial   string
	Version  semver
	Payload  any
}

func (d *serverDaemon) checkinHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var cd checkinData

	bodyBytes, err := io.ReadAll(r.Body)
	if checkError(err) {
		return
	}
	defer r.Body.Close()

	gob.Register(cd)

	b := bytes.NewReader(bodyBytes)
	gd := gob.NewDecoder(b)

	err = gd.Decode(&cd)
	if checkError(err) {
		return
	}

	var q string
	log.Printf("Agent (%d) with serial '%s', hyv_uuid '%s', and version '%s' (len: %d)", cd.ID, cd.Serial, cd.HyvID, cd.Version.String(), len(cd.Version.String()))

	if cd.Serial != "" {
		q = `SELECT id FROM agents WHERE hyv_uuid = $1`
		err = d.db.QueryRowContext(context.Background(), q, cd.HyvID).Scan(&cd.ID)
		if checkError(err) {
			return
		}

		q = `UPDATE agents SET version = $1 WHERE hyv_uuid = $2`
		_, err = d.db.ExecContext(context.Background(), q, cd.Version.String(), cd.HyvID)
		if checkError(err) {
			return
		}

		log.Printf("restoring ID %d to agent with hyv_uuid '%s'", cd.ID, cd.HyvID)
	}

	if cd.ID == 0 {
		log.Printf("Agent with serial '%s' and hyv_uuid '%s' checked in with invalid id 0", cd.Serial, cd.HyvID)

		return
		// q = "INSERT INTO agents (serial, os) VALUES ($1,$2) RETURNING id"
		// err = d.db.QueryRowContext(context.Background(), q, cd.Serial, cd.OS).Scan(&cd.ID)
		// if checkError(err) {
		// 	return
		// }
	}

	// log.Printf("SELECT input, c_uuid FROM commands WHERE agent_id = %d AND delivered_ts IS NULL AND scheduled_ts < NOW() ORDER BY scheduled_ts ASC", cd.ID)

	q = "SELECT input, c_uuid FROM commands WHERE agent_id = $1 AND delivered_ts IS NULL AND executed_ts IS NULL AND scheduled_ts < NOW() ORDER BY scheduled_ts ASC"
	rows, err := d.db.QueryContext(context.Background(), q, cd.ID)
	if checkError(err) {
		return
	}

	var cmds []Command
	for rows.Next() {
		c := Command{}
		err = rows.Scan(&c.Input, &c.UUID)
		if checkError(err) {
			return
		}

		q = "UPDATE commands SET delivered_ts = NOW() WHERE c_uuid = $1"
		_, err = d.db.ExecContext(context.Background(), q, c.UUID)
		if checkError(err) {
			return
		}

		cmds = append(cmds, c)
	}

	log.Printf("Found %d commands to run", len(cmds))

	var cr checkinResponse
	cr.ID = cd.ID

	var a agent
	d.agentsLocker.RLock()
	a, ok := d.agents[cr.ID]
	d.agentsLocker.RUnlock()

	if !ok {
		a, err = d.getAgentByID(cr.ID)
		if checkError(err) {
			return
		}
		d.agentsLocker.Lock()
		d.agents[cr.ID] = a
		d.agentsLocker.Unlock()
	}

	// log.Printf("Agent data: %#v", a)

	cr.StreamActivity = a.StreamingActivity

	d.currentAgentVersionLocker.RLock()
	currentAgentVersion = d.currentAgentVersion
	d.currentAgentVersionLocker.RUnlock()

	if cd.Version.isOlderThan(currentAgentVersion) {
		cr.Commands = append(cr.Commands, Command{Special: SpecialUpgrade})
	}

	cr.Commands = append(cr.Commands, cmds...)

	gob.Register(cr)

	ge := gob.NewEncoder(w)
	err = ge.Encode(cr)
	if checkError(err) {
		return
	}
}

func (d *serverDaemon) getLatestAgentVersion() {
	var (
		major int
		minor int
		patch int
	)

	err := d.db.QueryRowContext(context.Background(), `SELECT major, minor, patch FROM versions WHERE app = 'agent' 
		ORDER BY major desc, minor desc, patch desc LIMIT 1`).Scan(&major, &minor, &patch)
	if !errors.Is(err, sql.ErrNoRows) {
		if checkError(err) {
			return
		}
	}

	d.currentAgentVersionLocker.Lock()
	d.currentAgentVersion = semver{Major: major, Minor: minor, Patch: patch}
	d.currentAgentVersionLocker.Unlock()
}

func (d *serverDaemon) systemDataHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var cd checkinData
	bodyBytes, err := io.ReadAll(r.Body)
	if checkError(err) {
		return
	}

	b := bytes.NewReader(bodyBytes)
	gd := gob.NewDecoder(b)
	defer r.Body.Close()

	var aspo AppleSystemProfilerOutput
	var wsd *windowsSystemData
	gob.Register(aspo)
	gob.Register(wsd)

	err = gd.Decode(&cd)
	if checkError(err) {
		return
	}

	jsonBytes, err := json.Marshal(cd.Payload)
	if checkError(err) {
		return
	}

	log.Printf("Checkin with system data: %+v", cd)

	switch payload := cd.Payload.(type) {
	case AppleSystemProfilerOutput:
		// handle Apple
		aspo = cd.Payload.(AppleSystemProfilerOutput)
		log.Printf("Agent with serial '%s'", aspo.SPHardwareDataType[0].SerialNumber)
		log.Printf("Apple system data: %+v", payload)
		cd.Serial = aspo.SPHardwareDataType[0].SerialNumber
	case *windowsSystemData:
		// handle Windows
		wsd = cd.Payload.(*windowsSystemData)
		serial := wsd.BIOS.SerialNumber
		if wsd.Computer.Manufacturer == "QEMU" {
			serial = wsd.Product.UUID
		}
		cd.Serial = serial
		log.Printf("Agent with serial '%s'", serial)
		log.Printf("Windows system data: %+v", payload)
	default:
		log.Printf("Unknown payload type: %T", payload)
	}

	var assignedID int

	// q := "INSERT INTO agents (id, serial, os, hostname) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE os = ?, hostname = ?"
	q := `INSERT INTO agents (serial, os, host_name, system_data,hyv_uuid) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (hyv_uuid) DO UPDATE SET os = $6, host_name = $7, system_data = $8, hyv_uuid = $9  RETURNING id`
	err = d.db.QueryRowContext(context.Background(), q, cd.Serial, cd.OS, cd.Hostname, string(jsonBytes), cd.HyvID, cd.OS, cd.Hostname, string(jsonBytes), cd.HyvID).Scan(&assignedID)
	if checkError(err) {
		return
	}

	// assignedID, err := res.LastInsertId()
	// if checkError(err) {
	// 	return
	// }

	// if cd.ID == 0 && assignedID == 0 {
	// q = "SELECT id FROM agents WHERE serial = $1"
	// err = d.db.QueryRow(q, cd.Serial).Scan(&assignedID)
	// }

	q = "INSERT INTO checkins (agent_id, v_major, v_minor, v_patch) VALUES ($1,$2,$3,$4)"
	_, err = d.db.ExecContext(context.Background(), q, assignedID, cd.Version.Major, cd.Version.Minor, cd.Version.Patch)
	if checkError(err) {
		return
	}

	log.Printf("Corrected agent ID from %d to %d", cd.ID, assignedID)

	cr := checkinResponse{}
	cr.ID = assignedID

	gob.Register(cr)

	w.Header().Set("Content-Type", "application/octet-stream")

	ge := gob.NewEncoder(w)
	err = ge.Encode(cr)
	if checkError(err) {
		return
	}
}

func (d *serverDaemon) commandResultHandler(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	cmdResult := Command{}

	gob.Register(cmdResult)
	b := &bytes.Buffer{}

	bodyBytes, err := io.ReadAll(req.Body)
	if checkError(err) {
		return
	}

	_, err = b.Write(bodyBytes)
	if checkError(err) {
		return
	}

	gd := gob.NewDecoder(b)
	err = gd.Decode(&cmdResult)
	if checkError(err) {
		return
	}

	log.Printf("Updating command %s with output: '%s'", cmdResult.UUID, cmdResult.Output)

	q := "UPDATE commands SET output = $1, executed_ts = NOW() WHERE c_uuid = $2"
	_, err = d.db.ExecContext(context.Background(), q, cmdResult.Output, cmdResult.UUID)
	if checkError(err) {
		return
	}
}

type checkinResponse struct {
	ID             int
	Commands       []Command
	StreamActivity bool
}

type AppleSystemProfilerOutput struct {
	SPApplicationsDataType []struct {
		Name         string   `json:"_name"`
		ArchKind     string   `json:"arch_kind"`
		LastModified string   `json:"lastModified"`
		ObtainedFrom string   `json:"obtained_from"`
		Path         string   `json:"path"`
		SignedBy     []string `json:"signed_by,omitempty"`
		Version      string   `json:"version,omitempty"`
		Info         string   `json:"info,omitempty"`
	} `json:"SPApplicationsDataType"`
	SPConfigurationProfileDataType []struct {
		Items []struct {
			Items []struct {
				Name                             string `json:"_name"`
				SpconfigprofilePayloadData       string `json:"spconfigprofile_payload_data"`
				SpconfigprofilePayloadIdentifier string `json:"spconfigprofile_payload_identifier"`
				SpconfigprofilePayloadUUID       string `json:"spconfigprofile_payload_uuid"`
				SpconfigprofilePayloadVersion    int    `json:"spconfigprofile_payload_version"`
			} `json:"_items"`
			Name                             string `json:"_name"`
			SpconfigprofileDescription       string `json:"spconfigprofile_description"`
			SpconfigprofileInstallDate       string `json:"spconfigprofile_install_date"`
			SpconfigprofileOrganization      string `json:"spconfigprofile_organization"`
			SpconfigprofileProfileIdentifier string `json:"spconfigprofile_profile_identifier"`
			SpconfigprofileProfileUUID       string `json:"spconfigprofile_profile_uuid"`
			SpconfigprofileRemovalDisallowed string `json:"spconfigprofile_RemovalDisallowed"`
			SpconfigprofileVerificationState string `json:"spconfigprofile_verification_state"`
			SpconfigprofileVersion           int    `json:"spconfigprofile_version"`
		} `json:"_items"`
		Name string `json:"_name"`
	} `json:"SPConfigurationProfileDataType"`
	SPDisabledSoftwareDataType []struct {
		Name         string `json:"_name"`
		DisabledDate string `json:"disabledDate"`
		Reason       string `json:"reason"`
		Version      string `json:"version"`
	} `json:"SPDisabledSoftwareDataType"`
	SPDisplaysDataType []struct {
		Name                          string `json:"_name"`
		SpdisplaysMtlgpufamilysupport string `json:"spdisplays_mtlgpufamilysupport"`
		SpdisplaysNdrvs               []struct {
			Name                          string `json:"_name"`
			SpdisplaysDisplayProductID    string `json:"_spdisplays_display-product-id"`
			SpdisplaysDisplaySerialNumber string `json:"_spdisplays_display-serial-number"`
			SpdisplaysDisplayVendorID     string `json:"_spdisplays_display-vendor-id"`
			SpdisplaysDisplayWeek         string `json:"_spdisplays_display-week"`
			SpdisplaysDisplayYear         string `json:"_spdisplays_display-year"`
			SpdisplaysDisplayID           string `json:"_spdisplays_displayID"`
			SpdisplaysPixels              string `json:"_spdisplays_pixels"`
			SpdisplaysResolution          string `json:"_spdisplays_resolution"`
			SpdisplaysAmbientBrightness   string `json:"spdisplays_ambient_brightness,omitempty"`
			SpdisplaysMain                string `json:"spdisplays_main,omitempty"`
			SpdisplaysMirror              string `json:"spdisplays_mirror"`
			SpdisplaysOnline              string `json:"spdisplays_online"`
			SpdisplaysPixelresolution     string `json:"spdisplays_pixelresolution"`
			SpdisplaysResolution0         string `json:"spdisplays_resolution,omitempty"`
			SpdisplaysRotation            string `json:"spdisplays_rotation,omitempty"`
			SpdisplaysTelevision          string `json:"spdisplays_television,omitempty"`
			SpdisplaysConnectionType      string `json:"spdisplays_connection_type,omitempty"`
			SpdisplaysDisplayType         string `json:"spdisplays_display_type,omitempty"`
		} `json:"spdisplays_ndrvs"`
		SpdisplaysVendor string `json:"spdisplays_vendor"`
		SppciBus         string `json:"sppci_bus"`
		SppciCores       string `json:"sppci_cores"`
		SppciDeviceType  string `json:"sppci_device_type"`
		SppciModel       string `json:"sppci_model"`
	} `json:"SPDisplaysDataType"`
	SPEthernetDataType []struct {
		Name                     string `json:"_name"`
		SpethernetAvbSupport     string `json:"spethernet_avb_support"`
		SpethernetBSDDeviceName  string `json:"spethernet_BSD_Device_Name"`
		SpethernetBus            string `json:"spethernet_bus"`
		SpethernetDriver         string `json:"spethernet_driver"`
		SpethernetMacAddress     string `json:"spethernet_mac_address"`
		SpethernetProductName    string `json:"spethernet_product_name"`
		SpethernetProductID      string `json:"spethernet_product-id"`
		SpethernetUsbDeviceSpeed string `json:"spethernet_usb_device_speed"`
		SpethernetVendorName     string `json:"spethernet_vendor_name"`
		SpethernetVendorID       string `json:"spethernet_vendor-id"`
	} `json:"SPEthernetDataType"`
	SPFirewallDataType []struct {
		Name                     string `json:"_name"`
		SpfirewallGlobalstate    string `json:"spfirewall_globalstate"`
		SpfirewallLoggingenabled string `json:"spfirewall_loggingenabled"`
		SpfirewallStealthenabled string `json:"spfirewall_stealthenabled"`
	} `json:"SPFirewallDataType"`
	SPHardwareDataType []struct {
		Name                 string `json:"_name"`
		ActivationLockStatus string `json:"activation_lock_status"`
		BootRomVersion       string `json:"boot_rom_version"`
		ChipType             string `json:"chip_type"`
		MachineModel         string `json:"machine_model"`
		MachineName          string `json:"machine_name"`
		ModelNumber          string `json:"model_number"`
		NumberProcessors     string `json:"number_processors"`
		OsLoaderVersion      string `json:"os_loader_version"`
		PhysicalMemory       string `json:"physical_memory"`
		PlatformUUID         string `json:"platform_UUID"`
		ProvisioningUDID     string `json:"provisioning_UDID"`
		SerialNumber         string `json:"serial_number"`
	} `json:"SPHardwareDataType"`
	SPInstallHistoryDataType []struct {
		Name           string    `json:"_name"`
		InstallDate    time.Time `json:"install_date"`
		InstallVersion string    `json:"install_version,omitempty"`
		PackageSource  string    `json:"package_source"`
	} `json:"SPInstallHistoryDataType"`
	SPMemoryDataType []struct {
		DimmManufacturer string `json:"dimm_manufacturer"`
		DimmType         string `json:"dimm_type"`
		SPMemoryDataType string `json:"SPMemoryDataType"`
	} `json:"SPMemoryDataType"`
	SPNetworkDataType []struct {
		Name     string `json:"_name"`
		Ethernet struct {
			MACAddress   string `json:"MAC Address"`
			MediaOptions []any  `json:"MediaOptions"`
			MediaSubType string `json:"MediaSubType"`
		} `json:"Ethernet,omitempty"`
		Hardware  string `json:"hardware,omitempty"`
		Interface string `json:"interface,omitempty"`
		IPv4      struct {
			ConfigMethod string `json:"ConfigMethod"`
		} `json:"IPv4,omitempty"`
		IPv6 struct {
			ConfigMethod string `json:"ConfigMethod"`
		} `json:"IPv6,omitempty"`
		Proxies struct {
			ExceptionsList []string `json:"ExceptionsList"`
			FTPPassive     string   `json:"FTPPassive"`
		} `json:"Proxies,omitempty"`
		SpnetworkServiceOrder int    `json:"spnetwork_service_order"`
		Type                  string `json:"type"`
		Dhcp                  struct {
			DhcpDomainNameServers string `json:"dhcp_domain_name_servers"`
			DhcpLeaseDuration     int    `json:"dhcp_lease_duration"`
			DhcpMessageType       string `json:"dhcp_message_type"`
			DhcpRouters           string `json:"dhcp_routers"`
			DhcpServerIdentifier  string `json:"dhcp_server_identifier"`
			DhcpSubnetMask        string `json:"dhcp_subnet_mask"`
		} `json:"dhcp,omitempty"`
		DNS struct {
			SearchDomains   []string `json:"SearchDomains"`
			ServerAddresses []string `json:"ServerAddresses"`
		} `json:"DNS,omitempty"`
		IPAddress    []string `json:"ip_address,omitempty"`
		SleepProxies []struct {
			Name          string `json:"_name"`
			MarginalPower int    `json:"MarginalPower"`
			Metric        int    `json:"Metric"`
			Portability   int    `json:"Portability"`
			TotalPower    int    `json:"TotalPower"`
			Type          int    `json:"Type"`
		} `json:"sleep_proxies,omitempty"`
	} `json:"SPNetworkDataType"`
	SPNetworkVolumeDataType []struct {
		Name                       string `json:"_name"`
		SpnetworkvolumeAutomounted string `json:"spnetworkvolume_automounted"`
		SpnetworkvolumeFsmtnonname string `json:"spnetworkvolume_fsmtnonname"`
		SpnetworkvolumeFstypename  string `json:"spnetworkvolume_fstypename"`
		SpnetworkvolumeMntfromname string `json:"spnetworkvolume_mntfromname"`
	} `json:"SPNetworkVolumeDataType"`
	SPNVMeDataType []struct {
		Items []struct {
			Name              string `json:"_name"`
			BsdName           string `json:"bsd_name"`
			DetachableDrive   string `json:"detachable_drive"`
			DeviceModel       string `json:"device_model"`
			DeviceRevision    string `json:"device_revision"`
			DeviceSerial      string `json:"device_serial"`
			PartitionMapType  string `json:"partition_map_type"`
			RemovableMedia    string `json:"removable_media"`
			Size              string `json:"size"`
			SizeInBytes       int64  `json:"size_in_bytes"`
			SmartStatus       string `json:"smart_status"`
			SpnvmeTrimSupport string `json:"spnvme_trim_support"`
			Volumes           []struct {
				Name        string `json:"_name"`
				BsdName     string `json:"bsd_name"`
				Iocontent   string `json:"iocontent"`
				Size        string `json:"size"`
				SizeInBytes int    `json:"size_in_bytes"`
			} `json:"volumes"`
		} `json:"_items"`
		Name string `json:"_name"`
	} `json:"SPNVMeDataType"`
	SPPowerDataType []struct {
		Name                     string `json:"_name"`
		SppowerBatteryChargeInfo struct {
			SppowerBatteryAtWarnLevel   string `json:"sppower_battery_at_warn_level"`
			SppowerBatteryFullyCharged  string `json:"sppower_battery_fully_charged"`
			SppowerBatteryIsCharging    string `json:"sppower_battery_is_charging"`
			SppowerBatteryStateOfCharge int    `json:"sppower_battery_state_of_charge"`
		} `json:"sppower_battery_charge_info,omitempty"`
		SppowerBatteryHealthInfo struct {
			SppowerBatteryCycleCount            int    `json:"sppower_battery_cycle_count"`
			SppowerBatteryHealth                string `json:"sppower_battery_health"`
			SppowerBatteryHealthMaximumCapacity string `json:"sppower_battery_health_maximum_capacity"`
		} `json:"sppower_battery_health_info,omitempty"`
		SppowerBatteryModelInfo struct {
			PackLotCode                    string `json:"Pack Lot Code"`
			PCBLotCode                     string `json:"PCB Lot Code"`
			SppowerBatteryCellRevision     string `json:"sppower_battery_cell_revision"`
			SppowerBatteryDeviceName       string `json:"sppower_battery_device_name"`
			SppowerBatteryFirmwareVersion  string `json:"sppower_battery_firmware_version"`
			SppowerBatteryHardwareRevision string `json:"sppower_battery_hardware_revision"`
			SppowerBatterySerialNumber     string `json:"sppower_battery_serial_number"`
		} `json:"sppower_battery_model_info,omitempty"`
		ACPower struct {
			CurrentPowerSource                     string `json:"Current Power Source"`
			DiskSleepTimer                         int    `json:"Disk Sleep Timer"`
			DisplaySleepTimer                      int    `json:"Display Sleep Timer"`
			HibernateMode                          int    `json:"Hibernate Mode"`
			HighPowerMode                          int    `json:"HighPowerMode"`
			LowPowerMode                           int    `json:"LowPowerMode"`
			PrioritizeNetworkReachabilityOverSleep int    `json:"PrioritizeNetworkReachabilityOverSleep"`
			SleepOnPowerButton                     string `json:"Sleep On Power Button"`
			SystemSleepTimer                       int    `json:"System Sleep Timer"`
			WakeOnLAN                              string `json:"Wake On LAN"`
		} `json:"AC Power,omitempty"`
		BatteryPower struct {
			DiskSleepTimer                         int    `json:"Disk Sleep Timer"`
			DisplaySleepTimer                      int    `json:"Display Sleep Timer"`
			HibernateMode                          int    `json:"Hibernate Mode"`
			HighPowerMode                          int    `json:"HighPowerMode"`
			LowPowerMode                           int    `json:"LowPowerMode"`
			PrioritizeNetworkReachabilityOverSleep int    `json:"PrioritizeNetworkReachabilityOverSleep"`
			ReduceBrightness                       string `json:"ReduceBrightness"`
			SleepOnPowerButton                     string `json:"Sleep On Power Button"`
			SystemSleepTimer                       int    `json:"System Sleep Timer"`
			WakeOnLAN                              string `json:"Wake On LAN"`
		} `json:"Battery Power,omitempty"`
		SppowerUpsInstalled             string `json:"sppower_ups_installed,omitempty"`
		SppowerAcChargerFamily          string `json:"sppower_ac_charger_family,omitempty"`
		SppowerAcChargerFirmwareVersion string `json:"sppower_ac_charger_firmware_version,omitempty"`
		SppowerAcChargerHardwareVersion string `json:"sppower_ac_charger_hardware_version,omitempty"`
		SppowerAcChargerID              string `json:"sppower_ac_charger_ID,omitempty"`
		SppowerAcChargerManufacturer    string `json:"sppower_ac_charger_manufacturer,omitempty"`
		SppowerAcChargerName            string `json:"sppower_ac_charger_name,omitempty"`
		SppowerAcChargerSerialNumber    string `json:"sppower_ac_charger_serial_number,omitempty"`
		SppowerAcChargerWatts           string `json:"sppower_ac_charger_watts,omitempty"`
		SppowerBatteryChargerConnected  string `json:"sppower_battery_charger_connected,omitempty"`
		SppowerBatteryIsCharging        string `json:"sppower_battery_is_charging,omitempty"`
		Items                           []struct {
			Items []struct {
				AppPID      int    `json:"appPID"`
				Eventtype   string `json:"eventtype"`
				Scheduledby string `json:"scheduledby"`
				Time        string `json:"time"`
				UserVisible bool   `json:"UserVisible"`
			} `json:"_items"`
			Name string `json:"_name"`
		} `json:"_items,omitempty"`
	} `json:"SPPowerDataType"`
	SPPrefPaneDataType []struct {
		Name                 string `json:"_name"`
		SpprefpaneBundlePath string `json:"spprefpane_bundlePath"`
		SpprefpaneIdentifier string `json:"spprefpane_identifier"`
		SpprefpaneIsVisible  string `json:"spprefpane_isVisible"`
		SpprefpaneKind       string `json:"spprefpane_kind"`
		SpprefpaneSupport    string `json:"spprefpane_support"`
		SpprefpaneVersion    string `json:"spprefpane_version"`
	} `json:"SPPrefPaneDataType"`
	SPPrintersDataType []struct {
		Cupsversion string `json:"cupsversion"`
		Status      string `json:"status"`
	} `json:"SPPrintersDataType"`
	SPSecureElementDataType []struct {
		CtlFw              string `json:"ctl_fw"`
		CtlHw              string `json:"ctl_hw"`
		CtlInfo            string `json:"ctl_info"`
		CtlMw              string `json:"ctl_mw"`
		SeDevice           string `json:"se_device"`
		SeFw               string `json:"se_fw"`
		SeHw               string `json:"se_hw"`
		SeID               string `json:"se_id"`
		SeInRestrictedMode string `json:"se_in_restricted_mode"`
		SeInfo             string `json:"se_info"`
		SeOsVersion        string `json:"se_os_version"`
		SePlt              string `json:"se_plt"`
		SeProdSigned       string `json:"se_prod_signed"`
	} `json:"SPSecureElementDataType"`
	SPSoftwareDataType []struct {
		Name            string `json:"_name"`
		BootMode        string `json:"boot_mode"`
		BootVolume      string `json:"boot_volume"`
		KernelVersion   string `json:"kernel_version"`
		LocalHostName   string `json:"local_host_name"`
		OsVersion       string `json:"os_version"`
		SecureVM        string `json:"secure_vm"`
		SystemIntegrity string `json:"system_integrity"`
		Uptime          string `json:"uptime"`
		UserName        string `json:"user_name"`
	} `json:"SPSoftwareDataType"`
	SPStorageDataType []struct {
		Name             string `json:"_name"`
		BsdName          string `json:"bsd_name"`
		FileSystem       string `json:"file_system"`
		FreeSpaceInBytes int64  `json:"free_space_in_bytes"`
		IgnoreOwnership  string `json:"ignore_ownership"`
		MountPoint       string `json:"mount_point"`
		PhysicalDrive    struct {
			DeviceName       string `json:"device_name"`
			IsInternalDisk   string `json:"is_internal_disk"`
			MediaName        string `json:"media_name"`
			MediumType       string `json:"medium_type"`
			PartitionMapType string `json:"partition_map_type"`
			Protocol         string `json:"protocol"`
			SmartStatus      string `json:"smart_status"`
		} `json:"physical_drive,omitempty"`
		SizeInBytes int64  `json:"size_in_bytes"`
		VolumeUUID  string `json:"volume_uuid"`
		Writable    string `json:"writable"`
	} `json:"SPStorageDataType"`
	SPThunderboltDataType []struct {
		Name           string `json:"_name"`
		DeviceNameKey  string `json:"device_name_key"`
		DomainUUIDKey  string `json:"domain_uuid_key"`
		Receptacle1Tag struct {
			CurrentLinkWidthKey string `json:"current_link_width_key"`
			CurrentSpeedKey     string `json:"current_speed_key"`
			LinkStatusKey       string `json:"link_status_key"`
			ReceptacleIDKey     string `json:"receptacle_id_key"`
			ReceptacleStatusKey string `json:"receptacle_status_key"`
		} `json:"receptacle_1_tag"`
		RouteStringKey string `json:"route_string_key"`
		SwitchUIDKey   string `json:"switch_uid_key"`
		VendorNameKey  string `json:"vendor_name_key"`
	} `json:"SPThunderboltDataType"`
	SPUniversalAccessDataType []struct {
		Name         string `json:"_name"`
		Contrast     string `json:"contrast"`
		CursorMag    string `json:"cursor_mag"`
		Display      string `json:"display"`
		FlashScreen  string `json:"flash_screen"`
		KeyboardZoom string `json:"keyboardZoom"`
		MouseKeys    string `json:"mouse_keys"`
		ScrollZoom   string `json:"scrollZoom"`
		SlowKeys     string `json:"slow_keys"`
		StickyKeys   string `json:"sticky_keys"`
		Voiceover    string `json:"voiceover"`
		ZoomMode     string `json:"zoomMode"`
	} `json:"SPUniversalAccessDataType"`
	SPUSBDataType []struct {
		Items []struct {
			Name             string `json:"_name"`
			BcdDevice        string `json:"bcd_device"`
			BusPower         string `json:"bus_power"`
			BusPowerUsed     string `json:"bus_power_used"`
			DeviceSpeed      string `json:"device_speed"`
			ExtraCurrentUsed string `json:"extra_current_used"`
			LocationID       string `json:"location_id"`
			Manufacturer     string `json:"manufacturer"`
			ProductID        string `json:"product_id"`
			SerialNum        string `json:"serial_num"`
			VendorID         string `json:"vendor_id"`
		} `json:"_items"`
		Name           string `json:"_name"`
		HostController string `json:"host_controller"`
	} `json:"SPUSBDataType"`
}
