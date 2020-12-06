package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/jfrog/jfrog-cli-core/plugins/components"
	helpers "github.com/jfrog/jfrog-cli-plugin-template/utils"
)

func GetMetricsCommand() components.Command {
	return components.Command{
		Name:        "metrics",
		Description: "Get Metrics.",
		Aliases:     []string{"m"},
		Arguments:   getMetricsArguments(),
		Flags:       getMetricsFlags(),
		EnvVars:     getMetricsEnvVar(),
		Action: func(c *components.Context) error {
			return MetricsCmd(c)
		},
	}
}

func getMetricsArguments() []components.Argument {
	return []components.Argument{
		{
			Name:        "addressee",
			Description: "The name of the person you would like to greet.",
		},
	}
}

func getMetricsFlags() []components.Flag {
	return []components.Flag{
		components.BoolFlag{
			Name:         "raw",
			Description:  "Output straight from Artifactory",
			DefaultValue: false,
		},
		components.BoolFlag{
			Name:         "min",
			Description:  "Get minimum JSON from Artifactory",
			DefaultValue: false,
		},
		components.StringFlag{
			Name:         "repeat",
			Description:  "Greets multiple times.",
			DefaultValue: "1",
		},
	}
}

func getMetricsEnvVar() []components.EnvVar {
	return []components.EnvVar{
		{
			Name:        "Metrics_FROG_GREET_PREFIX",
			Default:     "A new greet from your plugin template: ",
			Description: "Adds a prefix to every greet.",
		},
	}
}

type MetricsConfiguration struct {
	addressee string
	raw       bool
	repeat    int
	prefix    string
	min       bool
}

func MetricsCmd(c *components.Context) error {

	config, err := helpers.GetConfig()
	if err != nil {
		return err
	}

	var conf = new(MetricsConfiguration)
	//conf.addressee = c.Arguments[0]
	conf.raw = c.GetBoolFlagValue("raw")

	if conf.raw {
		metricsRaw := helpers.GetMetricsDataRaw(config)
		fmt.Println(string(metricsRaw))
		return nil
	}

	conf.min = c.GetBoolFlagValue("min")

	if conf.min {
		//return json as is, no white space
		data, err := helpers.GetMetricsDataJSON(config, false)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	//else pretty print json
	data, err := helpers.GetMetricsDataJSON(config, true)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil

	repeat, err := strconv.Atoi(c.GetStringFlagValue("repeat"))
	if err != nil {
		return err
	}
	conf.repeat = repeat

	conf.prefix = os.Getenv("Metrics_FROG_GREET_PREFIX")
	if conf.prefix == "" {
		conf.prefix = "New greeting: "
	}

	//	log.Output(doGreet(conf))
	return nil
}
