package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/jfrog/jfrog-cli-core/plugins/components"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"

	helpers "github.com/jfrog/jfrog-cli-plugin-template/utils"

	"github.com/jfrog/jfrog-cli-core/utils/config"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"

	logFile "github.com/sirupsen/logrus"
)

//Data struct
type Data struct {
	Name   string    `json:"name"`
	Help   string    `json:"help"`
	Type   string    `json:"type"`
	Metric []Metrics `json:"metrics"`
}

//Metrics struct
type Metrics struct {
	TimestampMs string       `json:"timestamp_ms"`
	Value       string       `json:"value"`
	Labels      LabelsStruct `json:"labels,omitempty"`
}

//LabelsStruct struct
type LabelsStruct struct {
	Start  string `json:"start"`
	End    string `json:"end"`
	Status string `json:"status"`
	Type   string `json:"type"`
	Max    string `json:"max"`
	Pool   string `json:"pool"`
}

func GetGraphCommand() components.Command {
	return components.Command{
		Name:        "graph",
		Description: "Graph.",
		Aliases:     []string{"g"},
		Arguments:   getGraphArguments(),
		Flags:       getGraphFlags(),
		EnvVars:     getGraphEnvVar(),
		Action: func(c *components.Context) error {
			return GraphCmd(c)
		},
	}
}

func getGraphArguments() []components.Argument {
	return []components.Argument{
		{
			Name:        "addressee",
			Description: "The name of the person you would like to greet.",
		},
	}
}

func getGraphFlags() []components.Flag {
	return []components.Flag{
		components.BoolFlag{
			Name:         "shout",
			Description:  "Makes output uppercase.",
			DefaultValue: false,
		},
		components.StringFlag{
			Name:         "repeat",
			Description:  "Greets multiple times.",
			DefaultValue: "1",
		},
	}
}

func getGraphEnvVar() []components.EnvVar {
	return []components.EnvVar{
		{
			Name:        "Graph_FROG_GREET_PREFIX",
			Default:     "A new greet from your plugin template: ",
			Description: "Adds a prefix to every greet.",
		},
	}
}

type GraphConfiguration struct {
	addressee string
	shout     bool
	repeat    int
	prefix    string
}

func getServersIdAndDefault() ([]string, string, error) {
	allConfigs, err := config.GetAllArtifactoryConfigs()
	if err != nil {
		return nil, "", err
	}
	var defaultVal string
	var serversId []string
	for _, v := range allConfigs {
		if v.IsDefault {
			defaultVal = v.ServerId
		}
		serversId = append(serversId, v.ServerId)
	}
	return serversId, defaultVal, nil
}

func GraphCmd(c *components.Context) error {

	//TODO handle custom server id input
	serversIds, serverIdDefault, _ := getServersIdAndDefault()
	if len(serversIds) == 0 {
		return errorutils.CheckError(errors.New("no Artifactory servers configured. Use the 'jfrog rt c' command to set the Artifactory server details"))
	}

	//TODO handle if user is not admin

	//fmt.Print(serversIds, serverIdDefault)
	config, _ := config.GetArtifactorySpecificConfig(serverIdDefault, true, false)

	ping, _, _ := helpers.GetRestAPI("GET", true, config.Url+"api/system/ping", config.User, config.Password, "", nil, 1)
	if string(ping) != "OK" {
		logFile.Error("Artifactory is not up")
		return errors.New("Artifactory is not up")
	}

	if err := ui.Init(); err != nil {
		fmt.Printf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	o := widgets.NewParagraph()
	o.Title = "Meta statistics"
	o.Text = "Current time: " + time.Now().Format("2006.01.02 15:04:05")
	o.SetRect(0, 0, 77, 5)

	p := widgets.NewParagraph()
	p.Title = "Remote Connections"
	p.Text = "Initializing"
	p.SetRect(0, 6, 25, 11)

	q := widgets.NewParagraph()
	q.Title = "CPU Time (seconds)"
	q.Text = "Initializing"
	q.SetRect(26, 6, 51, 11)

	r := widgets.NewParagraph()
	r.Title = "Number of Metrics"
	r.Text = "Initializing"
	r.SetRect(52, 6, 77, 11)

	g2 := widgets.NewGauge()
	g2.Title = "Current Free Storage"
	g2.SetRect(0, 12, 50, 15)
	g2.Percent = 0
	g2.BarColor = ui.ColorYellow
	g2.LabelStyle = ui.NewStyle(ui.ColorBlue)
	g2.BorderStyle.Fg = ui.ColorWhite

	ui.Render(g2, o, p, q, r)

	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Second).C
	offSetCounter := 0
	for {
		select {
		case e := <-uiEvents:
			switch e.ID { // event string/identifier
			case "q", "<C-c>": // press 'q' or 'C-c' to quit
				return nil
			}

		// use Go's built-in tickers for updating and drawing data
		case <-ticker:
			var err error
			offSetCounter, err = drawFunction(config, g2, o, p, q, r, offSetCounter)
			if err != nil {
				return errorutils.CheckError(err)
			}

		}
	}
}

func drawFunction(config *config.ArtifactoryDetails, g2 *widgets.Gauge, o *widgets.Paragraph, p *widgets.Paragraph, q *widgets.Paragraph, r *widgets.Paragraph, offSetCounter int) (int, error) {
	responseTime := time.Now()
	data, lastUpdate, offset := getMetricsData(config, offSetCounter)

	var free, total *big.Float
	//var freeInt, totalInt int
	//maybe we can turn this into a hashtable for faster lookup
	for i := range data {

		var err error
		//TODO need logic to get more than 1 if there are multiple remote - there is a bug that halts the whole thing
		if data[i].Name == "jfrt_http_connections_max_total" {
			p.Text = data[i].Metric[0].Value + " " + data[i].Metric[0].Labels.Pool
		}
		if data[i].Name == "sys_cpu_totaltime_seconds" {
			q.Text = data[i].Metric[0].Value
		}
		if data[i].Name == "app_disk_free_bytes" {
			free, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				return 0, errorutils.CheckError(err)
			}
		}
		if data[i].Name == "app_disk_total_bytes" {
			total, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				return 0, errorutils.CheckError(err)
			}
		}

	}

	pctFreeSpace := new(big.Float).Mul(big.NewFloat(100), new(big.Float).Quo(free, total))

	pctFreeSpaceStr := pctFreeSpace.String()
	pctFreeSplit := strings.Split(pctFreeSpaceStr, ".")
	g2.Percent, _ = strconv.Atoi(pctFreeSplit[0])
	r.Text = strconv.Itoa(len(data))
	time.Now().Sub(responseTime)
	o.Text = "Current time: " + time.Now().Format("2006.01.02 15:04:05") + "\nLast updated: " + lastUpdate + " (" + strconv.Itoa(offset) + " seconds)\nResponse time: " + time.Now().Sub(responseTime).String()

	ui.Render(g2, o, p, q, r)
	return offset, nil
}

func getMetricsData(config *config.ArtifactoryDetails, counter int) ([]Data, string, int) {
	//log.Info("hello")
	//TODO check if token vs password apikey
	metrics, _, _ := helpers.GetRestAPI("GET", true, config.Url+"api/v1/metrics", config.User, config.Password, "", nil, 1)

	mfChan := make(chan *dto.MetricFamily, 1024)

	// Missing input means we are reading from an URL. stupid hack because Artifactory is missing a newline return
	file := string(metrics) + "\n"

	go func() {
		if err := prom2json.ParseReader(strings.NewReader(file), mfChan); err != nil {
			//fmt.Println("error reading metrics:", err)
			//fmt.Println(file)
			return
		}
	}()

	//TODO: Hella inefficient
	result := []*prom2json.Family{}
	for mf := range mfChan {
		result = append(result, prom2json.NewFamily(mf))
	}

	//pretty print
	//jsonText, err := json.MarshalIndent(result, "", "    ")

	jsonText, err := json.Marshal(result)
	if err != nil {
		fmt.Println(err)
		return nil, "", 0
	}

	var metricsData []Data
	err2 := json.Unmarshal(jsonText, &metricsData)
	if err2 != nil {
		fmt.Println(err2)
		return nil, "", 0
	}

	currentTime := time.Now()

	if len(metricsData) == 0 {
		counter = counter + 1
		currentTime = currentTime.Add(time.Second * -1 * time.Duration(counter))
	} else {
		counter = 0
	}
	return metricsData, currentTime.Format("2006.01.02 15:04:05"), counter
}

func doGreet2(c *GraphConfiguration) string {
	greet := c.prefix + "Graph " + c.addressee + "!\n"

	if c.shout {
		greet = strings.ToUpper(greet)
	}

	return strings.TrimSpace(strings.Repeat(greet, c.repeat))
}

func getMetrics() string {
	return ""
}
