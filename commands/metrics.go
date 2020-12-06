package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	helpers "github.com/jfrog/frogvision/utils"
	"github.com/jfrog/jfrog-cli-core/plugins/components"
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
			Name:        "list",
			Description: "list metrics.",
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
			Description:  "Get minimum JSON from Artifactory (no whitespace)",
			DefaultValue: false,
		},
	}
}

func getMetricsEnvVar() []components.EnvVar {
	return []components.EnvVar{}
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

	if len(c.Arguments) == 0 {
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
	}
	// probably not the right way to do it
	if len(c.Arguments) == 1 {
		var err error
		switch arg := c.Arguments[0]; arg {
		case "list":
			jsonText, err := helpers.GetMetricsDataJSON(config, false)
			if err != nil {
				return err
			}
			var metricsData []helpers.Data
			err2 := json.Unmarshal(jsonText, &metricsData)
			if err2 != nil {
				return err2
			}
			fmt.Println("Found", len(metricsData), "metrics")
			for i := range metricsData {
				fmt.Println(metricsData[i].Name)
			}
		case "linux":
			fmt.Println("Linux.")
		default:
			err = errors.New("Unrecognized argument:" + arg)
		}

		return err
	}
	return errors.New("Wrong number of arguments. Expected: 0 or 1, " + "Received: " + strconv.Itoa(len(c.Arguments)))

}
