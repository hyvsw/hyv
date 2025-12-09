package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

type navItem struct {
	Name string
	ID   int
	OS   string
}

func (d *serverDaemon) leftNavHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	log.Printf("leftNav requested")

	var navItems []navItem

	agents := d.getAgents(50, 0)

	for _, a := range agents {
		var ni navItem
		ni.ID = a.ID
		ni.Name = a.Name
		ni.OS = a.OS
		navItems = append(navItems, ni)
	}

	err := d.templates.ExecuteTemplate(w, "leftNav", navItems)
	if checkError(err) {
		return
	}
}

func (d *serverDaemon) viewAgentHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var err error
	id, err := strconv.Atoi(params.ByName("id"))
	if checkError(err) {
		return
	}

	log.Printf("agent %d requested", id)

	a, err := d.getAgentByID(id)
	if checkError(err) {
		return
	}

	switch a.OS {
	case "darwin":
		sData := darwinSystemData{}
		sData.AgentData = a

		err = json.Unmarshal([]byte(sData.AgentData.SystemData), &sData.SystemData)
		if checkError(err) {
			return
		}

		// parse out cpu count
		procString := sData.SystemData.SPHardwareDataType[0].NumberProcessors

		procString = strings.ReplaceAll(procString, "proc ", "")
		procCountStrings := strings.Split(procString, ":")
		sData.AgentData.CPUCountEfficiency, err = strconv.Atoi(procCountStrings[2])
		if checkError(err) {
			return
		}

		sData.AgentData.CPUCountPerformance, err = strconv.Atoi(procCountStrings[1])
		if checkError(err) {
			return
		}

		cs, err := d.getAgentCommands(id)
		if checkError(err) {
			return
		}

		sData.Commands = cs

		scripts, err := d.getScriptsForAgent(a.ID)
		if checkError(err) {
			return
		}

		sData.Scripts = scripts

		log.Printf(sData.SystemData.SPHardwareDataType[0].SerialNumber)

		var t time.Time
		q := "SELECT NOW()"
		err = d.db.QueryRow(q).Scan(&t)

		log.Printf("time: %s", t.String())

		q = "SELECT date_trunc('hour', ts) FROM checkins WHERE agent_id = $1 AND ts > NOW() - interval '7 days' GROUP BY date_trunc('hour', ts) ORDER BY date_trunc('hour', ts)"
		rows, err := d.db.QueryContext(context.Background(), q, id)
		if checkError(err) {
			return
		}

		// tz, _ := time.LoadLocation("America/Chicago")
		// tz, _ := time.LoadLocation("UTC")

		// var lastCheckin time.Time

		timestamps := make(map[int64]bool)
		for rows.Next() {
			var ts time.Time
			err = rows.Scan(&ts)
			if checkError(err) {
				return
			}

			timestamps[ts.UnixMilli()-(60*60*6*1000)] = true
			// lastCheckin = ts
		}

		// log.Printf("last checkin: %s", lastCheckin.String())

		type HCPoint struct {
			X int64 `json:"x"`
			Y int64 `json:"y"`
		}

		var randomizeSparkline bool
		var points []HCPoint
		for i := 6 * 24; i >= 0; i-- {
			// for each day we do not have a timestamp, it should indicate missing hour of checkins
			v := 0
			if _, ok := timestamps[time.Now().Truncate(time.Hour).Add(time.Hour*time.Duration(-i)).UnixMilli()]; ok {
				v = 1
			}

			if randomizeSparkline {
				if rand.Float32() > .7 {
					v = 0
				}
			}
			// if i > 3*24 {
			// v = 0
			// }

			points = append(points, HCPoint{X: time.Now().Truncate(time.Hour).Add(time.Duration(-i) * time.Hour).UnixMilli(), Y: int64(v)})
		}

		jsonBytes, err := json.Marshal(points)
		if checkError(err) {
			return
		}

		// jsonBytes = bytes.ReplaceAll(jsonBytes, []byte(`"x"`), []byte("x"))
		// jsonBytes = bytes.ReplaceAll(jsonBytes, []byte(`"y"`), []byte("y"))

		// log.Printf("points: '%s'", string(jsonBytes))

		checkinHistorySparkline := fmt.Sprintf(`<script>Highcharts.chart('checkin_history_sparkline', {
            chart: {
                type: 'area',
                margin: [0,0,0,0],
                backgroundColor: '#FF0000',
            },
            legend: {
                enabled: false,
            },
            credits: {
                enabled: false,
            },
            accessibility: {
                enabled: false,
            },
            title: {
                text: null
            },
            yAxis: {
                title: {
                    text: null,
                },
                labels: {
                    enabled: false
                },
                tickPositions: [0],
            },
            xAxis: {
                type: 'datetime',
                labels: {
                    enabled: false
                },
                tickPositions: [],
            },
            series: [{color: '#00FF00', data: %s}],
        })</script>`, string(jsonBytes))

		activityInstantaneousChart := fmt.Sprintf(`<script>Highcharts.chart('cpu_instantaneous_chart', {
                chart: {
                type: 'solidgauge',
                height: '110%%',
                events: {
                    render: renderIcons
                }
            },

            title: {
                text: 'Multiple KPI gauge',
                style: {
                    fontSize: '24px'
                }
            },

            tooltip: {
                borderWidth: 0,
                backgroundColor: 'none',
                shadow: false,
                style: {
                    fontSize: '16px'
                },
                valueSuffix: '%%',
                pointFormat: '{series.name}<br>' +
                    '<span style="font-size: 2em; color: {point.color}; ' +
                    'font-weight: bold">{point.y}</span>',
                positioner: function (labelWidth) {
                    return {
                        x: (this.chart.chartWidth - labelWidth) / 2,
                        y: (this.chart.plotHeight / 2) + 15
                    };
                }
            },

            pane: {
                startAngle: 0,
                endAngle: 360,
                background: [{ // Track for Conversion
                    outerRadius: '112%%',
                    innerRadius: '88%%',
                    backgroundColor: trackColors[0],
                    borderWidth: 0
                }, { // Track for Engagement
                    outerRadius: '87%%',
                    innerRadius: '63%%',
                    backgroundColor: trackColors[1],
                    borderWidth: 0
                }, { // Track for Feedback
                    outerRadius: '62%%',
                    innerRadius: '38%%',
                    backgroundColor: trackColors[2],
                    borderWidth: 0
                }]
            },

            yAxis: {
                min: 0,
                max: 100,
                lineWidth: 0,
                tickPositions: []
            },

            plotOptions: {
                solidgauge: {
                    dataLabels: {
                        enabled: false
                    },
                    linecap: 'round',
                    stickyTracking: false,
                    rounded: true
                }
            },

            series: [{
                name: 'Conversion',
                data: [{
                    color: Highcharts.getOptions().colors[0],
                    radius: '112%%',
                    innerRadius: '88%%',
                    y: 80
                }],
                custom: {
                    icon: 'filter',
                    iconColor: '#303030'
                }
            }, {
                name: 'Engagement',
                data: [{
                    color: Highcharts.getOptions().colors[1],
                    radius: '87%%',
                    innerRadius: '63%%',
                    y: 65
                }],
                custom: {
                    icon: 'comments-o',
                    iconColor: '#ffffff'
                }
            }, {
                name: 'Feedback',
                data: [{
                    color: Highcharts.getOptions().colors[2],
                    radius: '62%%',
                    innerRadius: '38%%',
                    y: 50
                }],
                custom: {
                    icon: 'commenting-o',
                    iconColor: '#303030'
                }
            }]
        })</script>`)

		sData.CheckinHistorySparkline = template.HTML(checkinHistorySparkline)
		sData.ActivityInstantaneousChart = template.HTML(activityInstantaneousChart)

		// sData.systemData = a.SystemData.(AppleSystemProfilerOutput)
		b := bytes.NewBuffer(nil)
		err = d.templates.ExecuteTemplate(b, "agent-darwin", sData)
		if checkError(err) {
			return
		}

		responseBytes := b.Bytes()

		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Length", strconv.Itoa(len(responseBytes)))
		w.Write(responseBytes)

	case "windows":
	}

	d.agentStartStreamActivityHandler(w, r, params)
}

func (d *serverDaemon) sendCommandHandler(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	req.ParseForm()

	id, err := strconv.Atoi(params.ByName("id"))
	if checkError(err) {
		return
	}

	input := req.Form.Get("input")

	cUUID, err := uuid.NewUUID()
	if checkError(err) {
		return
	}

	log.Printf("Recieved command '%s' for agent %d", input, id)

	var agentID int

	q := "INSERT INTO commands (agent_id, input, c_uuid, scheduled_ts) VALUES ($1, $2, $3, NOW()) RETURNING id"
	err = d.db.QueryRowContext(context.Background(), q, id, input, cUUID.String()).Scan(&agentID)
	if checkError(err) {
		return
	}

	time.Sleep(time.Millisecond * 200)

	params = append(params, httprouter.Param{Key: "agentID", Value: strconv.Itoa(id)})

	d.commandHistoryForAgentHandler(w, req, params)
}

func (d *serverDaemon) commandOutputRefreshHandler(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	id, err := strconv.Atoi(params.ByName("commandID"))
	if checkError(err) {
		return
	}

	var co Command
	q := "SELECT id, COALESCE(output,''), executed_ts FROM commands WHERE id = $1"
	err = d.db.QueryRowContext(context.Background(), q, id).Scan(&co.ID, &co.Output, &co.ExecutedTS)
	if checkError(err) {
		return
	}

	if co.ExecutedTS.Valid {
		co.Executed = true
	}

	b := bytes.NewBuffer(nil)
	err = d.templates.ExecuteTemplate(b, "command-output", co)
	if checkError(err) {
		return
	}

	responseBytes := b.Bytes()

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Content-Length", strconv.Itoa(len(responseBytes)))
	w.Write(responseBytes)
}

type Script struct {
	ID          int
	Name        string
	Created     time.Time
	Modified    time.Time
	Notes       string
	Creator     int
	Description string
}

type ScriptParameters struct {
	ScriptID  int
	ID        int
	Name      string
	ValueType string
}

func (d *serverDaemon) getScriptsForAgent(id int) ([]Script, error) {
	q := "SELECT id, name, created_ts, modified_ts, notes, creator_id, description FROM script_library WHERE os = (SELECT os FROM agents WHERE id = $1)"
	rows, err := d.db.QueryContext(context.Background(), q, id)
	if checkError(err) {
		return nil, err
	}

	var scripts []Script

	for rows.Next() {
		s := Script{}
		err = rows.Scan(&s.ID, &s.Name, &s.Created, &s.Modified, &s.Notes, &s.Creator, &s.Description)
	}

	return scripts, nil
}
