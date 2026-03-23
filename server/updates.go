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

type semver struct {
	Major int
	Minor int
	Patch int
}

func (v semver) isOlderThan(sv semver) bool {
	// log.Printf("Checking if '%s' is older than '%s'", v.String(), sv.String())
	if v.Major != sv.Major {
		return v.Major < sv.Major
	}
	if v.Minor != sv.Minor {
		return v.Minor < sv.Minor
	}
	return v.Patch < sv.Patch
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
		d.currentAgentVersionLocker.RLock()
		if version.isOlderThan(d.currentUpdaterVersion) {
			w.WriteHeader(201)
		}
		d.currentAgentVersionLocker.RUnlock()
	case "agent":
		d.currentAgentVersionLocker.RLock()
		if version.isOlderThan(d.currentAgentVersion) {
			w.WriteHeader(201)
		}
		d.currentAgentVersionLocker.RUnlock()
	}
}

func (d *serverDaemon) buildAppHandler(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	app := params.ByName("App")
	vMajor, err := strconv.Atoi(params.ByName("vMajor"))
	if checkError(err) {
		return
	}
	vMinor, err := strconv.Atoi(params.ByName("vMinor"))
	if checkError(err) {
		return
	}
	vPatch, err := strconv.Atoi(params.ByName("vPatch"))
	if checkError(err) {
		return
	}

	d.updateLatestAppVersion(app, vMajor, vMinor, vPatch)
	d.getLatestAgentVersion()

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
		`-X main.controlServerHost=%s -X main.controlServerPort=%s -X main.versionMajorStr=%s -X main.versionMinorStr=%s -X main.versionPatchStr=%s`,
		os.Getenv("HYV_CONTROL_SERVER_HOST"),
		os.Getenv("HYV_CONTROL_SERVER_PORT"),
		strconv.Itoa(vMajor),
		strconv.Itoa(vMinor),
		strconv.Itoa(vPatch),
	)

	var bytesWritten int

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
		appPath, err := filepath.Abs(fmt.Sprintf("../%s", app))
		if checkError(err) {
			return
		}
		cmd.Dir = appPath

		// log.Printf("app directory: '%s'", cmd.Dir)

		// log.Printf("build: %#v", cmd.Args)

		out, err := cmd.CombinedOutput()
		if checkError(err) {
			log.Printf("out: %s", string(out))
			n, err := w.Write([]byte(fmt.Sprintf("error while building '%s' for '%s' for '%s", app, oa.os, oa.arch)))
			if checkError(err) {
				return
			}
			n, err = w.Write(out)
			if checkError(err) {
				return
			}
			bytesWritten += n
			w.Write([]byte("\n"))
			return
		}

		if len(out) > 0 {
			log.Printf("build completed with output: %s", string(out))
		} else {
			log.Printf("successfully built '%s' for '%s' for '%s'", app, oa.os, oa.arch)
		}
	}
	if bytesWritten == 0 {
		w.Write([]byte("No errors during builds"))
	}
}
