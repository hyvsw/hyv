package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/kardianos/service"
)

var (
	versionMajor = 0
	versionMinor = 0
	versionPatch = 5
)

type updaterDaemon struct {
	daemonCfg     *service.Config
	daemon        service.Service
	hc            http.Client
	controlServer string
	programUrl    url.URL
	installPath   string
	version       semver
}

type agentDaemon struct {
	ID                    int
	daemonCfg             *service.Config
	daemon                service.Service
	hc                    http.Client
	controlServer         string
	programUrl            url.URL
	installPath           string
	version               semver
	debug                 bool
	cmdr                  string
	lastSystemDataCheckin time.Time
}

func (v semver) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v semver) JSON() string {
	return fmt.Sprintf(`{"Major": %d, "Minor": %d, "Patch": %d}`, v.Major, v.Minor, v.Patch)
}

type semver struct {
	Major int
	Minor int
	Patch int
}

// these control server variables are set with the build script using ldflags
var (
	ControlServerHost string
	ControlServerPort string
)

func newDaemon() *updaterDaemon {
	d := &updaterDaemon{}
	d.hc.Timeout = time.Minute * 2
	d.hc.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	d.controlServer = fmt.Sprintf("%s:%s", ControlServerHost, ControlServerPort)

	return d
}

var currentAgentVersion semver = semver{Major: 0, Minor: 0, Patch: 2}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// log.Printf("host: '%s', port: '%s'", ControlServerHost, ControlServerPort)

	// largely just going to sit around and wait until a newer agent is available
	// checking every 24 hours for new agent
	// localhost listener allows agent to poke and perform on-demand agent updates
	ud := newDaemon()
	ud.programUrl.Scheme = "http"
	ud.programUrl.Host = ud.controlServer
	ud.programUrl.Path = fmt.Sprintf("/static/downloads/%s/%s/hyv_updater", runtime.GOOS, runtime.GOARCH)
	ud.daemonCfg = getPlatformUpdaterConfig()
	var err error
	ud.daemon, err = service.New(ud, ud.daemonCfg)
	if checkError(err) {
		return
	}

	ad := &agentDaemon{}
	ad.programUrl.Scheme = "http"
	ad.programUrl.Host = ud.controlServer
	ad.programUrl.Path = fmt.Sprintf("/static/downloads/%s/%s/hyv_agent", runtime.GOOS, runtime.GOARCH)
	ad.daemonCfg = getPlatformAgentConfig()
	ad.daemon, err = service.New(ad, ad.daemonCfg)
	if checkError(err) {
		return
	}

	if service.Interactive() {
		err = ad.daemon.Stop()
		if checkError(err) {
			// return
		}

		err = ad.daemon.Uninstall()
		if checkError(err) {
			// return
		}

		err = download(ad.programUrl.String(), ad.daemonCfg.Executable)
		if checkError(err) {
			// return
		}

		err = ad.daemon.Install()
		if checkError(err) {
			// return
		}

		err = ad.daemon.Start()
		if checkError(err) {
			return
		}
		log.Printf("Service started")
		return
	} else {

		go func() {
			err = ad.checkForUpdates()
			if checkError(err) {
				// return
			}
			t := time.NewTicker(time.Hour * 25)
			for {
				select {
				case <-t.C:
					err = ad.checkForUpdates()
					if checkError(err) {
						// return
					}
				}
			}
		}()

		err = ud.daemon.Run()
		if checkError(err) {
			return
		}
	}
}

func (d *updaterDaemon) Start(s service.Service) error {
	return nil
}

func (d *updaterDaemon) Stop(s service.Service) error {
	return nil
}

func (d *agentDaemon) Start(s service.Service) error {
	return nil
}

func (d *agentDaemon) Stop(s service.Service) error {
	return nil
}

func download(endpoint, toPath string) (err error) {
	hc := &http.Client{Timeout: time.Minute * 2}

	log.Printf("Attempting to download from thing at place '%s'", endpoint)
	resp, err := hc.Get(endpoint)
	if checkError(err) {
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if checkError(err) {
		return
	}
	defer resp.Body.Close()

	log.Printf("Received %s file", BytesToHuman(int64(len(bodyBytes))))

	err = os.MkdirAll(filepath.Dir(toPath), 0o700)
	if checkError(err) {
		return
	}

	f, err := os.OpenFile(toPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0o700)
	if checkError(err) {
		return
	}

	n, err := f.Write(bodyBytes)
	if checkError(err) {
		return
	}

	log.Printf("Wrote %s to file at '%s'", BytesToHuman(int64(n)), toPath)

	err = f.Sync()
	if checkError(err) {
		return
	}

	err = f.Close()
	if checkError(err) {
		return
	}

	return nil
}

func (d *updaterDaemon) installAgent() (err error) {
	err = d.daemon.Install()
	if checkError(err) {
		if errors.Is(err, service.ErrNotInstalled) {
		}
		return
	}

	err = d.daemon.Start()
	if checkError(err) {
		return
	}
	return
}

func (d *agentDaemon) checkForUpdates() (err error) {
	maxRetryWait, err := time.ParseDuration("8h")
	if checkError(err) {
		return
	}
	retryWait, err := time.ParseDuration("10s")
	if checkError(err) {
		return
	}
	var retryStage int64

	var agentNotResponsive bool

	log.Printf("Checking agent version")

	resp, err := d.hc.Get("https://localhost:22130/api/v1/version")
	if checkError(err) {
		agentNotResponsive = true
		// return
	}

	var sv semver
	if !agentNotResponsive {
		bodyBytes, err := io.ReadAll(resp.Body)
		if checkError(err) {
			return err
		}
		resp.Body.Close()

		err = json.Unmarshal(bodyBytes, &sv)
		if checkError(err) {
			return err
		}

		u := fmt.Sprintf("%s://%s/api/v1/check/version/agent/%d/%d/%d", d.programUrl.Scheme, d.programUrl.Host, sv.Major, sv.Minor, sv.Patch)
		log.Printf("Checking if version is outdated with control server at '%s'", u)

		resp, err = d.hc.Get(u)
		if checkError(err) {
			return err
		}

		log.Printf("Server responded with statuscode %d", resp.StatusCode)

		if resp.StatusCode != http.StatusCreated {
			return err
		}

	}

	for {
		if retryStage == 0 {
			log.Printf("Trying download of agent")
		} else {
			log.Printf("Retrying (try %d) download of agent", retryStage)
		}
		err = download(d.programUrl.String(), d.daemonCfg.Executable)
		if err == nil {
			log.Printf("Agent download successful")

			err = d.daemon.Stop()
			if checkError(err) {
				// return
			}

			err = d.daemon.Uninstall()
			if checkError(err) {
				// return
			}

			err = d.daemon.Install()
			if checkError(err) {
				// return
			}

			err = d.daemon.Start()
			if checkError(err) {
				return
			}
			return
		}
		retryStage++

		// exponential backoff
		retryWait = time.Duration(retryWait.Nanoseconds() ^ retryStage)
		if retryWait.Nanoseconds() > maxRetryWait.Nanoseconds() {
			retryWait = maxRetryWait
		}
		log.Printf("Waiting %s to retry", retryWait.String())
		time.Sleep(retryWait)
	}
}

func (d *updaterDaemon) uninstallAgent() (err error) {
	err = d.daemon.Stop()
	if checkError(err) {
		// return
	}

	err = d.daemon.Uninstall()
	if checkError(err) {
		return
	}
	return
}
