package main

import (
	"os"

	"github.com/jfrog/frogvision/commands"
	helpers "github.com/jfrog/frogvision/utils"
	"github.com/jfrog/jfrog-cli-core/plugins"
	"github.com/jfrog/jfrog-cli-core/plugins/components"
	"github.com/prometheus/common/log"
)

func main() {

	// You could set this to any `io.Writer` such as a file (declared in rest.go)
	file, err := os.OpenFile(helpers.LogFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		helpers.LogRestFile.Out = file
	} else {
		log.Info("Failed to log to file, using default stderr")
	}
	//TODO FLAGS
	plugins.PluginMain(getApp())

}

func getApp() components.App {
	app := components.App{}
	app.Name = "frogvision"
	app.Description = "Easily graph anyone."
	app.Version = "v0.1.0"
	app.Commands = getCommands()
	return app
}

func getCommands() []components.Command {
	return []components.Command{
		commands.GetHelloCommand(),
		commands.GetGraphCommand(),
		commands.GetMetricsCommand(),
	}
}
