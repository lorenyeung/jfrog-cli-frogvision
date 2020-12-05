package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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

	//fmt.Print(serversIds, serverIdDefault)
	config, _ := config.GetArtifactorySpecificConfig(serverIdDefault, true, false)

	if err := ui.Init(); err != nil {
		fmt.Printf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	p := widgets.NewParagraph()
	p.Text = "Hello World!"
	p.SetRect(0, 0, 25, 5)

	ui.Render(p)

	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Second).C
	for {
		select {
		case e := <-uiEvents:
			switch e.ID { // event string/identifier
			case "q", "<C-c>": // press 'q' or 'C-c' to quit
				return nil
			}

		// use Go's built-in tickers for updating and drawing data
		case <-ticker:
			drawFunction(config, p)
		}
	}
}

func drawFunction(config *config.ArtifactoryDetails, p *widgets.Paragraph) {
	data := getMetricsData(config)

	for i := range data {
		if data[i].Name == "sys_cpu_totaltime_seconds" {
			p.Text = data[i].Metric[0].Value
		}
	}
	ui.Render(p)
}

func getMetricsData(config *config.ArtifactoryDetails) []Data {
	//TODO check if token vs password apikey
	metrics, _, _ := helpers.GetRestAPI("GET", true, config.Url+"api/v1/metrics", config.User, config.Password, "", nil, 1)

	mfChan := make(chan *dto.MetricFamily, 1024)

	// Missing input means we are reading from an URL. stupid hack because Artifactory is missing a newline return
	file := string(metrics) + "\n"

	go func() {
		if err := prom2json.ParseReader(strings.NewReader(file), mfChan); err != nil {
			fmt.Println("error reading metrics:", err)
			os.Exit(1)
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
		os.Exit(1)
	}

	var metricsData []Data
	err2 := json.Unmarshal(jsonText, &metricsData)
	if err2 != nil {
		fmt.Println(err2)
		os.Exit(1)
	}

	return metricsData
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
