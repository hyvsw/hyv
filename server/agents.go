package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
)

type agent struct {
	ID                   int
	Version              string
	ClientID             int
	Name                 string
	Alias                sql.NullString
	ModelID              sql.NullInt64
	MfgID                sql.NullInt64
	Serial               string
	Deleted              time.Time
	OS                   string
	Arch                 string
	SystemData           string
	CPUCountPerformance  int
	CPUCountEfficiency   int
	StreamingActivity    bool
	LatestActivity       Activity
	LatestActivityLocker *sync.RWMutex
}

func (d *serverDaemon) getAgents(limit, skip int) []*agent {
	q := "SELECT id, client_id, host_name, os, serial, system_data, streaming_activity FROM agents WHERE id NOT IN (select id FROM deleted_agents) ORDER BY host_name asc LIMIT $1 OFFSET $2"
	rows, err := d.db.QueryContext(context.Background(), q, limit, skip)
	if checkError(err) {
		return nil
	}

	var agents []*agent
	for rows.Next() {
		a := &agent{}
		err = rows.Scan(&a.ID, &a.ClientID, &a.Name, &a.OS, &a.Serial, &a.SystemData, &a.StreamingActivity)
		if checkError(err) {
			return nil
		}

		agents = append(agents, a)
	}

	log.Printf("Returning %d agents", len(agents))

	return agents
}

func (d *serverDaemon) getAgentByID(id int) (agent, error) {
	var a agent

	a, ok := d.agents[id]
	if !ok {
		a.LatestActivityLocker = &sync.RWMutex{}
		q := "SELECT id, client_id, host_name, serial, os, system_data, streaming_activity FROM agents WHERE id = $1"
		err := d.db.QueryRowContext(context.Background(), q, id).Scan(&a.ID, &a.ClientID, &a.Name, &a.Serial, &a.OS, &a.SystemData, &a.StreamingActivity)
		if checkError(err) {
			if errors.Is(err, sql.ErrNoRows) {
				return a, fmt.Errorf("agent '%d' does not exist: %w", id, err)
			}
			return a, err
		}

		// todo fix Apple

		switch a.OS {
		case "darwin":
			// handle Apple
			var data AppleSystemProfilerOutput
			err = json.Unmarshal([]byte(a.SystemData), &data)
			if checkError(err) {
				return a, err
			}
			// log.Printf("SystemData: %#v", data)
			cpuCountStrings := strings.Split(strings.TrimPrefix(data.SPHardwareDataType[0].NumberProcessors, "proc "), ":")
			a.CPUCountEfficiency, err = strconv.Atoi(cpuCountStrings[2])
			if checkError(err) {
				return a, err
			}
			a.CPUCountPerformance, err = strconv.Atoi(cpuCountStrings[1])
			if checkError(err) {
				return a, err
			}
		case "windows":
			// handle Windows
			var data windowsSystemData
			err = json.Unmarshal([]byte(a.SystemData), &data)
			if checkError(err) {
				return a, err
			}

		case "linux":
		default:
			log.Printf("Unknown OS: %s", a.OS)
		}

		// log.Printf("Agent data: %#v", a)
		d.agentsLocker.Lock()
		d.agents[id] = a
		d.agentsLocker.Unlock()
	}

	return a, nil
}

func (d *serverDaemon) commandHistoryForAgentHandler(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	id, err := strconv.Atoi(params.ByName("agentID"))
	if checkError(err) {
		return
	}

	a, err := d.getAgentByID(id)
	if checkError(err) {
		return
	}

	sData := darwinSystemData{}
	sData.AgentData = a

	cs, err := d.getAgentCommands(id)
	if checkError(err) {
		return
	}

	sData.Commands = cs

	b := bytes.NewBuffer(nil)
	err = d.templates.ExecuteTemplate(b, "command_window", sData)
	if checkError(err) {
		return
	}

	responseBytes := b.Bytes()

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Content-Length", strconv.Itoa(len(responseBytes)))
	w.Write(responseBytes)
}
