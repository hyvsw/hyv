package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
)

func (d *agentDaemon) checkin() {
	var data checkinData

	data.ID = d.ID
	data.Version = d.version

	sd, err := d.getSystemData()
	if checkError(err) {
		return
	}
	if sd != nil {
		data.Serial = sd.BIOS.SerialNumber
	}
	if sd.Computer.Manufacturer == "QEMU" {
		data.serial = sd.Product.UUID
	}

	b := &bytes.Buffer{}
	ge := gob.NewEncoder(b)
	err = ge.Encode(data)
	if checkError(err) {
		return
	}

	resp, err := d.hc.Post(fmt.Sprintf("%s://%s/%s", d.programUrl.Scheme, d.controlServer, checkinPath), "application/octet-stream", b)
	if checkError(err) {
		return
	}
	defer resp.Body.Close()
}

func (d *agentDaemon) getSystemData() (data *windowsSystemData, err error) {
	// use powershell to collect system data
	//
	jsonData, err := run(`[ordered]@{
  BIOS = Get-CimInstance Win32_BIOS | Select-Object SerialNumber, SMBIOSBIOSVersion
  Computer = Get-CimInstance Win32_ComputerSystem | Select-Object Manufacturer, Model
  BaseBoard = Get-CimInstance Win32_BaseBoard | Select-Object Product, Manufacturer, SerialNumber
  OS = Get-CimInstance Win32_OperatingSystem | Select-Object Caption, Version, OSArchitecture
  Product = Get-CimInstance Win32_ComputerSystemProduct | Select-Object UUID
} | ConvertTo-Json -Depth 3`)

	sd := &windowsSystemData{}

	err = json.Unmarshal([]byte(jsonData), sd)
	if checkError(err) {
		return nil, err
	}

	d.systemData = sd

	return sd, nil
}

func (d *agentDaemon) sendSystemData() {
	var data checkinData
	data.ID = d.ID

	output, err := run("hostname")
	if checkError(err) {
		return
	}

	d.hostname = output

	data.Hostname = output
	data.OS = runtime.GOOS

	// start := time.Now()
	log.Printf("Reading system data...")
	d.systemData, err = readSystemData()
	if checkError(err) {
		return
	}
	data.Payload = d.systemData
	data.Version = d.version
	// sd, err := d.getSystemData()
	// if checkError(err) {
	// 	return
	// }
	// if sd == nil {
	// 	log.Printf("System data is nil")
	// 	return
	// }

	// log.Printf("Got system data (took %s): %+v", time.Since(start).String(), d.getSystemData())
	// log.Printf("System data: %+v", d.getSystemData())

	b := &bytes.Buffer{}
	gob.Register(data)
	gob.Register(data.Payload)
	ge := gob.NewEncoder(b)
	err = ge.Encode(data)
	if checkError(err) {
		return
	}

	resp, err := d.hc.Post(fmt.Sprintf("%s://%s/%s", d.programUrl.Scheme, d.controlServer, systemDataPath), "application/octet-stream", b)
	if checkError(err) {
		return
	}
	defer resp.Body.Close()

	var cr checkinResponse

	gob.Register(cr)
	gd := gob.NewDecoder(resp.Body)

	err = gd.Decode(&cr)
	if checkError(err) {
		return
	}

	log.Printf("Response: %#v", cr)

	d.ID = cr.ID
}

type checkinResponse struct {
	ID             int
	Commands       []Command
	StreamActivity bool
}

func readSystemData() (*windowsSystemData, error) {
	// jsonData, err := run(fmt.Sprintf("get-computerinfo | convertto-json"))
	// if checkError(err) {
	// 	return nil, err
	// }

	jsonData, err := run(`[ordered]@{
  BIOS = Get-CimInstance Win32_BIOS | Select-Object SerialNumber, SMBIOSBIOSVersion
  Computer = Get-CimInstance Win32_ComputerSystem | Select-Object Manufacturer, Model
  BaseBoard = Get-CimInstance Win32_BaseBoard | Select-Object Product, Manufacturer, SerialNumber
  OS = Get-CimInstance Win32_OperatingSystem | Select-Object Caption, Version, OSArchitecture
  Product = Get-CimInstance Win32_ComputerSystemProduct | Select-Object UUID
} | ConvertTo-Json -Depth 3`)

	si := &windowsSystemData{}

	err = json.Unmarshal([]byte(jsonData), si)
	if checkError(err) {
		return nil, err
	}

	return si, nil
}

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

// Actually, use get-computerinfo | convertto-json

// use msinfo32 /nfo C:\Windows\temp\output.xml
// or use powershell: get-ciminstance -class win32_operatingsystem | convertto-json > C:\Windows\temp\output.json
