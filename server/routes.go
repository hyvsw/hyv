package main

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

func (d *serverDaemon) bindRoutes() {
	d.router.GET("/", d.baseHandler)
	d.router.GET("/login", d.loginHandler)
	d.router.GET("/logout", d.logoutHandler)
	d.router.POST("/login", d.loginHandler)
	d.router.GET("/static/*path", d.staticHandler)

	d.router.POST("/api/v1/user/add/:name", d.userAddHandler)

	d.router.GET("/api/v1/parts/leftNav", d.leftNavHandler)
	d.router.GET("/api/v1/parts/agent/:id", d.viewAgentHandler)
	d.router.GET("/api/v1/parts/commands/agent/:id", d.commandHistoryForAgentHandler)

	d.router.POST("/api/v1/sendCommand/:id", d.sendCommandHandler)
	d.router.POST("/api/v1/checkin", d.checkinHandler)
	d.router.POST("/api/v1/sendSystemData", d.systemDataHandler)

	d.router.GET("/api/v1/check/version/:App/:Major/:Minor/:Patch", d.versionCheckHandler)
	d.router.GET("/api/v1/build/:App", d.buildAppHandler)

	d.router.POST("/api/v1/sendCommandResult", d.commandResultHandler)

	d.router.GET("/api/v1/agent/:id/stream/activity", d.agentStartStreamActivityHandler)
	d.router.POST("/api/v1/agent/:id/stream/activity", d.agentStreamActivityMomentHandler)
	d.router.DELETE("/api/v1/agent/:id/stream/activity", d.agentEndStreamActivityHandler)

	// d.router.GET("/api/v1/agent/:id/stream/read/moment", d.agentStreamActivityMomentReaderHandler)
	d.router.GET("/api/v1/agent/:id/stream/read/:ActivityName", d.agentStreamActivityReaderHandler)

	// Results for 1 command
	d.router.GET("/api/v1/parts/commands/history/:agentID", d.commandHistoryForAgentHandler)
	// All command results
	d.router.GET("/api/v1/parts/command/output/:commandID", d.commandOutputRefreshHandler)
}

func (d *serverDaemon) baseHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	log.Printf("Returning index page")

	var data struct{}

	err := d.templates.ExecuteTemplate(w, "index", data)
	if checkError(err) {
		return
	}
}

func (d *serverDaemon) userAddHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	r.ParseForm()
	pass := r.Form.Get("pass")

	if r.Form.Get("userAddKey") != os.Getenv("HYV_USER_ADD_KEY") {
		log.Printf("failed to add user '%s': unauthorized", params.ByName("name"))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Printf("Adding user '%s'", params.ByName("name"))

	salt := os.Getenv("PASS_SALT")

	rounds, err := strconv.Atoi(os.Getenv("PASS_ROUNDS"))
	if checkError(err) {
		return
	}

	if rounds < 1 {
		rounds = 1
	}

	sha := sha512.New()

	for i := 0; i <= rounds; i++ {
		_, err = sha.Write([]byte(pass + salt))
		if checkError(err) {
			return
		}
	}

	pass = hex.EncodeToString(sha.Sum(nil))

	_, err = d.db.ExecContext(context.Background(), "INSERT INTO users (username, password_hash) VALUES ($1, $2)", params.ByName("name"), pass)
	if checkError(err) {
		return
	}

	log.Printf("Created user '%s'", params.ByName("name"))
}

func (d *serverDaemon) authHandler(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		cookie, err := r.Cookie("session")
		if checkError(err) {
			return
		}

		log.Printf("attempting to authenticate session '%s'", cookie.Value)

		var uuid string
		var username string
		err = d.db.QueryRowContext(context.Background(), "SELECT uuid, username FROM user_sessions LEFT JOIN users ON (user_sessions.user_id = users.id) WHERE uuid = $1", cookie.Value).Scan(&uuid, &username)
		if checkError(err) {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			// w.Header().Set("Location", "/login")
			// w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}

		if uuid != "" {
			log.Printf("user '%s' has valid session id: '%s'", username, uuid)
			h(w, r, params)
			return
		}

		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
}

func (d *serverDaemon) logoutHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	cookie, err := r.Cookie("session")
	if checkError(err) {
		return
	}

	sessionUUID := cookie.Value

	if sessionUUID == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	log.Printf("logging out session '%s'", sessionUUID)

	_, err = d.db.ExecContext(context.Background(), "UPDATE user_sessions SET logged_out_ts = NOW() WHERE uuid = $1", sessionUUID)
	if checkError(err) {
		return
	}

	cookie.Value = ""
	cookie.MaxAge = 0

	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/login", http.StatusSeeOther)
	// w.Header().Set("Location", "/login")
	// w.WriteHeader(http.StatusTemporaryRedirect)
}

func (d *serverDaemon) loginHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if r.Method == http.MethodGet {
		log.Printf("Returning login page")
		var data struct{}
		err := d.templates.ExecuteTemplate(w, "login", data)
		if checkError(err) {
			return
		}
		return
	}

	r.ParseForm()
	user := r.Form.Get("user")
	pass := r.Form.Get("pass")

	log.Printf("authenticating user '%s'", user)

	salt := os.Getenv("PASS_SALT")

	rounds, err := strconv.Atoi(os.Getenv("PASS_ROUNDS"))
	if checkError(err) {
		return
	}

	if rounds < 1 {
		rounds = 1
	}

	sha := sha512.New()

	for i := 0; i <= rounds; i++ {
		_, err = sha.Write([]byte(pass + salt))
		if checkError(err) {
			return
		}
	}

	pass = hex.EncodeToString(sha.Sum(nil))

	var authenticated bool

	var userID int
	err = d.db.QueryRow("SELECT count(id) FROM users WHERE username = $1 AND password_hash = $2", user, pass).Scan(&userID)
	if checkError(err) {
		return
	}

	if userID > 0 {
		authenticated = true
	}

	if authenticated {
		// create session UUID, send to browser
		theUUID, err := uuid.NewV7()
		if checkError(err) {
			return
		}

		_, err = d.db.ExecContext(context.Background(), "INSERT INTO user_sessions (user_id, uuid) VALUES ($1, $2)", userID, theUUID.String())
		if checkError(err) {
			return
		}

		http.SetCookie(w, &http.Cookie{Name: "session", Domain: os.Getenv("HYV_DOMAIN"), Value: theUUID.String()})

		log.Printf("Successfully authenticated '%s'", user)
		log.Printf("Returning base page")

		http.Redirect(w, r, "/", http.StatusSeeOther)

	} else {
		log.Printf("WARNING: User '%s' failed to authenticate", user)
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}
}

func (d *serverDaemon) staticHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	fName := fmt.Sprintf("./static%s", params.ByName("path"))

	log.Printf("Handling static request '%s'", fName)

	fileBytes, err := os.ReadFile(fName)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if strings.HasSuffix(fName, "js") {
		w.Header().Set("Content-Type", "application/javascript")
	}
	if strings.HasSuffix(fName, "css") {
		w.Header().Set("Content-Type", "text/css")
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(fileBytes)))

	w.Write(fileBytes)
}
