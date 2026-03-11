package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/kardianos/service"
)

func (d *agentDaemon) commandProcessor() {
	var err error

	for {
		select {
		case c := <-d.commandChan:

			if c.Special > 0 {
				log.Print("Executing special %v", c.Special)
				d.executeSpecial(c.Special)
			} else {
				log.Printf("Received command %s: '%s'", c.UUID, c.Input)
				c.Output, err = run(c.Input)
				if checkError(err) {
					// Do something smart?
				}
				d.returnCommandResult(c)
			}
		}
	}
}

func (d *agentDaemon) executeSpecial(sc specialCommand) error {
	switch sc {
	case specialCmdUpgrade:
		log.Printf("Executing agent upgrade")
		err := d.upgradeAgent()
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *agentDaemon) upgradeAgent() (err error) {
	log.Printf("Attempting to download updater from '%s'", d.programUrl.String())
	resp, err := d.hc.Get(d.programUrl.String())
	if checkError(err) {
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if checkError(err) {
		return
	}
	defer resp.Body.Close()

	log.Printf("Received %s file", BytesToHuman(int64(len(bodyBytes))))

	ud := newDaemon()
	ud.daemonCfg = getPlatformUpdaterConfig()

	f, err := os.Create(ud.daemonCfg.Executable)
	if checkError(err) {
		return
	}

	n, err := f.Write(bodyBytes)
	if checkError(err) {
		return
	}

	log.Printf("Wrote %s to file at '%s'", BytesToHuman(int64(n)), ud.daemonCfg.Executable)

	err = f.Sync()
	if checkError(err) {
		return
	}

	err = f.Close()
	if checkError(err) {
		return
	}

	log.Printf("new daemon")

	ud.daemon, err = service.New(ud, ud.daemonCfg)
	if checkError(err) {
		return
	}

	// log.Printf("daemon stop")

	err = ud.daemon.Stop()
	if checkError(err) {
		return
	}
	// log.Printf("daemon uninstall")

	err = ud.daemon.Uninstall()
	if checkError(err) {
		return
	}
	// log.Printf("daemon install")

	err = ud.daemon.Install()
	if checkError(err) {
		return
	}

	// log.Printf("daemon start")

	err = ud.daemon.Start()
	if checkError(err) {
		return
	}

	log.Printf("updater daemon started")

	return nil
}

func (d *agentDaemon) returnCommandResult(c Command) {
	b := &bytes.Buffer{}

	gob.Register(c)

	ge := gob.NewEncoder(b)
	err := ge.Encode(c)
	if checkError(err) {
		return
	}

	log.Printf("Returning command result for UUID %s: '%s'", c.UUID, c.Output)

	_, err = d.hc.Post(fmt.Sprintf("%s://%s/%s", d.programUrl.Scheme, d.programUrl.Host, commandResultPath), "application/octet-stream", b)
	if checkError(err) {
		return
	}
}
