package main

import (
	"log"
	"time"
)

const (
	checkinPath              string = "/api/v1/checkin"
	systemDataPath                  = "/api/v1/sendSystemData"
	commandResultPath               = "/api/v1/sendCommandResult"
	streamActivityMomentPath        = "/api/v1/agent/%d/stream/activity/"
)

func (d *agentDaemon) checkinProcessor() {
	t := time.NewTicker(time.Minute)
	systemDataCheckinTicker := time.NewTimer(time.Until(d.lastSystemDataCheckin.Add(time.Hour)))

	for {
		select {
		case <-t.C:
			// log.Printf("Checking in with id (%d) and serial '%s', with version %s", d.ID, d.getSystemData().SPHardwareDataType[0].SerialNumber, d.version.string())
			d.checkin()
		case <-systemDataCheckinTicker.C:
			log.Printf("Sending in system data...")
			systemDataCheckinTicker = time.NewTimer(time.Until(time.Now().Add(time.Hour)))
			d.lastSystemDataCheckin = time.Now()
			go d.sendSystemData()
		}
	}
}

type checkinData struct {
	ID       int
	Hostname string
	OS       string
	Serial   string
	Version  semver
	Payload  any
}
