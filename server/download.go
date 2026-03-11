package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

func (d *serverDaemon) downloadAppHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	appArg := params.ByName("App")
	osArg := params.ByName("Os")
	archArg := params.ByName("Arch")

	log.Printf("request for os '%s', arch '%s', app '%s'", osArg, archArg, appArg)

	extensionArg := ""
	if osArg == "windows" {
		extensionArg = "exe"
	}
	f, err := os.Open(fmt.Sprintf("static/downloads/%s/%s/%s/hyv_%s%s", appArg, osArg, archArg, appArg, extensionArg))
	if checkError(err) {
		return
	}

	n, err := io.Copy(w, f)
	if checkError(err) {
		return
	}

	log.Printf("wrote %d bytes to %s", n, r.RemoteAddr)
}
