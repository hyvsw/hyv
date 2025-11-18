package main

import (
	"log"
	"time"
)

func (d *serverDaemon) startServer() {

	// certCacheDir := "/etc/letsencrypt"
	//
	// m := &autocert.Manager{
	// 	Cache:      autocert.DirCache(certCacheDir),
	// 	Prompt:     autocert.AcceptTOS,
	// 	HostPolicy: autocert.HostWhitelist("hyv.infinity.local"),
	// }

	d.hs.Addr = ":2213"
	d.hs.Handler = d.router
	d.hs.ReadTimeout = 5 * time.Minute
	d.hs.WriteTimeout = 5 * time.Minute
	d.hs.MaxHeaderBytes = 1 << 20
	// d.hs.TLSConfig = m.TLSConfig()

	// log.Fatal(d.hs.ListenAndServeTLS("", ""))
	log.Fatal(d.hs.ListenAndServe())
}
