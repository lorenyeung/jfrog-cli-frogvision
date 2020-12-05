package commands

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/jfrog/jfrog-cli-core/plugins/components"
	"github.com/jfrog/jfrog-client-go/utils/log"

	helpers "github.com/jfrog/jfrog-cli-plugin-template/utils"
)

func GetGraphCommand() components.Command {
	return components.Command{
		Name:        "Graph",
		Description: "Says Graph.",
		Aliases:     []string{"hi"},
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

func GraphCmd(c *components.Context) error {

	metrics, _, _ := helpers.GetRestAPI("GET", true, "http://localhost:8081/artifactory/api/v1/metrics", "admin", "password", "", nil, 1)

	fmt.Println(string(metrics))
	if err := ui.Init(); err != nil {
		fmt.Printf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	p := widgets.NewParagraph()
	p.Text = "Hello World!"
	p.SetRect(0, 0, 25, 5)

	ui.Render(p)

	for e := range ui.PollEvents() {
		fmt.Println("test")
		if e.Type == ui.KeyboardEvent {
			break
		}
	}

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
