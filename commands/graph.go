package commands

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jfrog/jfrog-cli-core/plugins/components"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"

	helpers "github.com/jfrog/jfrog-cli-plugin-template/utils"

	"github.com/jfrog/jfrog-cli-core/utils/config"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
)

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

	//TODO check if token vs password apikey
	metrics, _, _ := helpers.GetRestAPI("GET", true, config.Url+"api/v1/metrics", config.User, config.Password, "", nil, 1)
	//fmt.Println(string(metrics))
	// if err := ui.Init(); err != nil {
	// 	fmt.Printf("failed to initialize termui: %v", err)
	// }
	// defer ui.Close()

	// p := widgets.NewParagraph()
	// p.Text = "Hello World!"
	// p.SetRect(0, 0, 25, 5)

	// ui.Render(p)

	// for e := range ui.PollEvents() {
	// 	fmt.Println("test")
	// 	if e.Type == ui.KeyboardEvent {
	// 		break
	// 	}
	// }

	mfChan := make(chan *dto.MetricFamily, 1024)

	// Missing input means we are reading from an URL.
	file := string(metrics) + "\n"

	go func() {
		if err := prom2json.ParseReader(strings.NewReader(file), mfChan); err != nil {
			fmt.Println("error reading metrics:", err)
			os.Exit(1)
		}
	}()
	// go func() {
	// 	err := prom2json.FetchMetricFamilies(arg, mfChan, nil)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		os.Exit(1)
	// 	}
	// }()

	result := []*prom2json.Family{}
	for mf := range mfChan {
		result = append(result, prom2json.NewFamily(mf))
	}
	jsonText, err := json.MarshalIndent(result, "", "    ")

	fmt.Println(string(jsonText))

	if len(c.Arguments) != 1 {
		return errors.New("Wrong number of arguments. Expected: 1, " + "Received: " + strconv.Itoa(len(c.Arguments)))
	}
	var conf = new(GraphConfiguration)
	conf.addressee = c.Arguments[0]
	conf.shout = c.GetBoolFlagValue("shout")

	repeat, err := strconv.Atoi(c.GetStringFlagValue("repeat"))
	if err != nil {
		return err
	}
	conf.repeat = repeat

	conf.prefix = os.Getenv("Graph_FROG_GREET_PREFIX")
	if conf.prefix == "" {
		conf.prefix = "New greeting: "
	}

	log.Output(doGreet2(conf))
	return nil
}

func makeTransport(
	certificate string, key string,
	skipServerCertCheck bool,
) (*http.Transport, error) {
	var transport *http.Transport
	if certificate != "" && key != "" {
		cert, err := tls.LoadX509KeyPair(certificate, key)
		if err != nil {
			return nil, err
		}
		tlsConfig := &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: skipServerCertCheck,
		}
		tlsConfig.BuildNameToCertificate()
		transport = &http.Transport{TLSClientConfig: tlsConfig}
	} else {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: skipServerCertCheck},
		}
	}
	return transport, nil
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
