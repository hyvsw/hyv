package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

var (
	currentAgentVersion   semver = semver{Major: 0, Minor: 0, Patch: 62}
	currentUpdaterVersion semver = semver{Major: 0, Minor: 0, Patch: 5}
)

type semver struct {
	Major int
	Minor int
	Patch int
}

func (v semver) isOlderThan(sv semver) bool {
	if v.Major < sv.Major {
		return true
	}
	if v.Minor < sv.Minor {
		return true
	}
	if v.Patch < sv.Patch {
		return true
	}
	return false
}

func (v semver) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (d *serverDaemon) versionCheckHandler(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	version := semver{}
	var err error

	version.Major, err = strconv.Atoi(params.ByName("Major"))
	if checkError(err) {
		return
	}

	version.Minor, err = strconv.Atoi(params.ByName("Minor"))
	if checkError(err) {
		return
	}

	version.Patch, err = strconv.Atoi(params.ByName("Patch"))
	if checkError(err) {
		return
	}

	switch params.ByName("App") {
	case "updater":
		if version.isOlderThan(currentUpdaterVersion) {
			w.WriteHeader(201)
		}
	case "agent":
		if version.isOlderThan(currentAgentVersion) {
			w.WriteHeader(201)
		}
	}
}

func (d *serverDaemon) buildAppHandler(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	app := params.ByName("App")

	switch app {
	case "agent":
		log.Printf("building %s", app)
	case "updater":
		log.Printf("building %s", app)
	default:
		log.Printf("unexpected app name '%s'", app)
		return
	}
	// cmd = exec.Command("./build.sh")
	type osarch struct {
		os   string
		arch string
	}

	var osarches []osarch
	osarches = append(osarches, osarch{os: "darwin", arch: "arm64"})
	osarches = append(osarches, osarch{os: "darwin", arch: "amd64"})
	osarches = append(osarches, osarch{os: "linux", arch: "amd64"})
	osarches = append(osarches, osarch{os: "linux", arch: "arm64"})
	osarches = append(osarches, osarch{os: "windows", arch: "amd64"})
	osarches = append(osarches, osarch{os: "windows", arch: "arm64"})

	ldflags := fmt.Sprintf(
		`-X main.controlServerHost=%s -X main.controlServerPort=%s`,
		os.Getenv("HYV_CONTROL_SERVER_HOST"),
		os.Getenv("HYV_CONTROL_SERVER_PORT"),
	)

	for _, oa := range osarches {

		extension := ""
		if oa.os == "windows" {
			extension = ".exe"
		}

		staticDestDir, err := filepath.Abs(fmt.Sprintf("./static/downloads/%s/%s/hyv_%s%s", oa.os, oa.arch, app, extension))
		if checkError(err) {
			return
		}

		if err := os.MkdirAll(filepath.Dir(staticDestDir), 0o755); err != nil {
			log.Printf("failed to create output directory: %w", err)
			return
		}

		cmd := exec.Command("/usr/local/go/bin/go", "build", "-buildvcs=false", "-o", staticDestDir, "-ldflags", ldflags)
		cmd.Env = append(os.Environ(), fmt.Sprintf("GOOS=%s", oa.os), fmt.Sprintf("GOARCH=%s", oa.arch))
		agentPath, err := filepath.Abs("../agent")
		if checkError(err) {
			return
		}
		cmd.Dir = agentPath

		log.Printf("build: %#v", cmd.Args)

		out, err := cmd.CombinedOutput()
		if checkError(err) {
			log.Printf("out: %s", string(out))
			return
		}

		d.getLatestAgentVersion()

		log.Printf("done building: %s", string(out))
	}
}
