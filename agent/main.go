package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/kardianos/service"
)

var (
	versionMajor = 0
	versionMinor = 2
	versionPatch = 0
)

func (v semver) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v semver) JSON() string {
	return fmt.Sprintf(`{"Major": %d, "Minor": %d, "Patch": %d}`, v.Major, v.Minor, v.Patch)
}

type agentDaemon struct {
	ID                    int
	HyvID                 string
	hostname              string
	daemonCfg             *service.Config
	daemon                service.Service
	hc                    http.Client
	hs                    http.Server
	programUrl            url.URL
	installPath           string
	version               semver
	debug                 bool
	controlServer         string
	lastSystemDataCheckin time.Time
	systemData            any
	commandChan           chan Command

	streamingActivity bool
	doneStreamingChan chan int
}

// these control server variables are set with the build script using ldflags
var (
	controlServerHost string
	controlServerPort string
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if runtime.GOOS == "windows" {
		logFilePath := "C:\\ProgramData\\hyv\\hyv_agent.log"
		logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if checkError(err) {
			return
		}
		log.SetOutput(logFile)
	}

	d := newDaemon()
	log.SetPrefix("v" + d.version.String() + ": ")

	d.commandChan = make(chan Command, 50)
	d.doneStreamingChan = make(chan int, 1)

	log.Printf("Agent version %s starting...", d.version.String())

	var err error
	d.HyvID, err = getOrCreateAgentID()
	if checkError(err) {
		return
	}

	go d.commandProcessor()

	if service.Interactive() {
		d.debug = true
		d.deployInstaller()
		return
	}

	go d.runAgent()

	err = d.daemon.Run()
	if checkError(err) {
		return
	}
}

func (d *agentDaemon) Start(s service.Service) error {
	return nil
}

func (d *agentDaemon) Stop(s service.Service) error {
	return nil
}

func (d *agentDaemon) runAgent() {
	log.Printf("Agent running")
	go d.checkinProcessor()

	d.hs = http.Server{
		Addr: ":22130",
	}

	log.Printf("Binding routes...")

	d.bindRoutes()

	log.Printf("Starting agent daemon server...")
	err := d.hs.ListenAndServe()
	if checkError(err) {
		return
	}

	log.Printf("Shutting down...")
}

func (d *agentDaemon) bindRoutes() {
	http.DefaultServeMux.HandleFunc("/api/v1/version", d.versionEchoHandler)
}

func (d *agentDaemon) versionEchoHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("content-type", "application/json")
	w.Write([]byte(d.version.JSON()))
}
