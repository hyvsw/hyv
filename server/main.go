package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	// _ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5"
	"github.com/julienschmidt/httprouter"
)

type serverDaemon struct {
	hc                        http.Client
	hs                        http.Server
	router                    *httprouter.Router
	templates                 *template.Template
	db                        *sql.DB
	currentAgentVersion       semver
	currentAgentVersionLocker *sync.RWMutex

	agentsLocker *sync.RWMutex
	agents       map[int]agent
}

func parseTemplates() *template.Template {
	fm := template.FuncMap{"divide": func(a, b int) float64 {
		return (float64(a) / float64(b)) * 100
	}}
	templ := template.New("").Funcs(fm)
	err := filepath.Walk("./templates", func(path string, info os.FileInfo, err error) error {

		if info.IsDir() {
			return nil
		}
		_, err = templ.ParseFiles(path)
		if err != nil {
			log.Println(err)
		}

		return err
	})

	if err != nil {
		panic(err)
	}

	return templ
}

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	d := &serverDaemon{}

	d.db = InitializePgSQLDatabase(os.Getenv("PGSQL_HOST"), os.Getenv("PGSQL_DB"), os.Getenv("PGSQL_USER"), url.QueryEscape(os.Getenv("PGSQL_PASS")))
	d.router = httprouter.New()
	d.templates = parseTemplates()
	d.agents = make(map[int]agent)
	d.agentsLocker = &sync.RWMutex{}
	d.currentAgentVersionLocker = &sync.RWMutex{}

	// d.hs = http.Server{
	// 	Addr:    ":2213",
	// 	Handler: d.router,
	// }
	d.getLatestAgentVersion()

	log.Printf("Binding routes...")

	d.bindRoutes()

	log.Printf("Starting server...")
	d.startServer()
	// err := d.hs.ListenAndServe()
	// if checkError(err) {
	// 	return
	// }

	log.Printf("Shutting down...")

}
