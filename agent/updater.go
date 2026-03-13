package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kardianos/service"
)

type semver struct {
	Major int
	Minor int
	Patch int
}

func newDaemon() *agentDaemon {
	var err error
	versionMajor, err = strconv.Atoi(versionMajorStr)
	if checkError(err) {
		return nil
	}
	versionMinor, err = strconv.Atoi(versionMinorStr)
	if checkError(err) {
		return nil
	}
	versionPatch, err = strconv.Atoi(versionPatchStr)
	if checkError(err) {
		return nil
	}

	d := &agentDaemon{}
	d.hc.Timeout = time.Second * 30
	d.hc.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	d.controlServer = fmt.Sprintf("%s:%s", controlServerHost, controlServerPort)
	d.programUrl.Scheme = "http"
	d.programUrl.Host = d.controlServer
	d.programUrl.Path = fmt.Sprintf("/api/v1/download/agent/%s/%s", runtime.GOOS, runtime.GOARCH)

	d.version = semver{Major: versionMajor, Minor: versionMinor, Patch: versionPatch}

	d.daemonCfg = getPlatformAgentConfig()
	d.daemon, err = service.New(d, d.daemonCfg)
	if checkError(err) {
		return d
	}

	return d
}

func (d *agentDaemon) deployInstaller() {
	log.Printf("deploying agent installer")
	var err error

	err = d.daemon.Install()
	if checkError(err) {
		if !strings.Contains(err.Error(), "Init already exists") &&
			!strings.Contains(err.Error(), "service hyv_agent already exists") {
			return
		}
	}

	err = d.daemon.Start()
	if checkError(err) {
		return
	}
}

func (d *agentDaemon) downloadUpdater() (err error) {
	ud := newDaemon()
	ud.programUrl.Path = fmt.Sprintf("/api/v1/download/updater/%s/%s", runtime.GOOS, runtime.GOARCH)
	ud.daemonCfg = getPlatformUpdaterConfig()

	log.Printf("Attempting to download agent from '%s'", ud.programUrl.String())
	resp, err := d.hc.Get(ud.programUrl.String())
	if checkError(err) {
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if checkError(err) {
		return
	}
	defer resp.Body.Close()

	log.Printf("Received %s file", BytesToHuman(int64(len(bodyBytes))))

	f, err := os.Create(ud.installPath)
	if checkError(err) {
		return
	}

	n, err := f.Write(bodyBytes)
	if checkError(err) {
		return
	}

	log.Printf("Wrote %s to file at '%s'", BytesToHuman(int64(n)), d.installPath)

	err = f.Sync()
	if checkError(err) {
		return
	}

	err = f.Close()
	if checkError(err) {
		return
	}

	log.Printf("Stopping existing hyv_updater daemon")
	err = ud.daemon.Stop()
	if checkError(err) {
		// return
	}

	log.Printf("Installing hyv_updater daemon")
	err = ud.daemon.Install()
	if checkError(err) {
		return
	}

	log.Printf("Starting hyv_updater daemon")
	err = ud.daemon.Start()
	if checkError(err) {
		return
	}

	return nil
}
