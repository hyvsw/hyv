package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/uuid"
)

func getOrCreateAgentID() (string, error) {
	path := `/etc/hyv/agent.id`
	id, err := os.ReadFile(path)
	if err == nil && len(id) > 0 {
		return string(id), nil
	}
	newID := uuid.New().String()
	err = os.MkdirAll(filepath.Dir(path), 0o644)
	if checkError(err) {
		return "", err
	}
	err = os.WriteFile(path, []byte(newID), 0o644)
	if checkError(err) {
		return "", err
	}
	return newID, nil
}

type checkinResponse struct {
	ID             int
	Commands       []Command
	StreamActivity bool
}

type LinuxSystemData struct{}

func (d *agentDaemon) getSystemData() *LinuxSystemData {
	return &LinuxSystemData{}
}

func (d *agentDaemon) checkin() {
	var data checkinData

	data.ID = d.ID
	data.Version = d.version

	sd := d.getSystemData()
	if sd != nil {
		// data.Serial = sd.SPHardwareDataType[0].SerialNumber
	}

	gob.Register(data)

	b := &bytes.Buffer{}
	ge := gob.NewEncoder(b)
	err := ge.Encode(data)
	if checkError(err) {
		return
	}

	resp, err := d.hc.Post(fmt.Sprintf("%s://%s/%s", d.programUrl.Scheme, d.controlServer, checkinPath), "application/octet-stream", b)
	if !errors.Is(err, io.EOF) && checkError(err) {
		return
	}
	defer resp.Body.Close()

	cr := checkinResponse{}
	gob.Register(cr)

	gd := gob.NewDecoder(resp.Body)
	err = gd.Decode(&cr)
	if !errors.Is(err, io.EOF) && checkError(err) {
		return
	}

	d.ID = cr.ID

	if cr.Commands != nil {
		log.Printf("Received commands from server: %#v", cr)
	} else {
		log.Printf("Received no commands from server")
	}

	if cr.StreamActivity {
		if !d.streamingActivity {
			d.streamingActivity = true
			log.Printf("Starting stream of activity data")
			// go d.streamActivity()
		}
	}

	if !cr.StreamActivity {
		if d.streamingActivity {
			d.streamingActivity = false
			log.Printf("Ending stream of activity data")
			d.doneStreamingChan <- 1
		}
	}

	for _, cmd := range cr.Commands {
		d.commandChan <- cmd
	}
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
	// d.systemData, err = readSystemData()
	if checkError(err) {
		return
	}
	data.Payload = d.systemData
	data.Version = d.version
	sd := d.getSystemData()
	if sd == nil {
		log.Printf("System data is nil")
		return
	}

	b := &bytes.Buffer{}
	gob.Register(data)
	gob.Register(*sd)
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

	cr := checkinResponse{}
	gob.Register(cr)

	gd := gob.NewDecoder(resp.Body)
	err = gd.Decode(&cr)
	if checkError(err) {
		return
	}

	log.Printf("Response: %#v", cr)

	d.ID = cr.ID

	d.checkin()
}
