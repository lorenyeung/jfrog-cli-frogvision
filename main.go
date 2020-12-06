package main

import (
	"github.com/jfrog/frogvision/commands"
	"github.com/jfrog/jfrog-cli-core/plugins"
	"github.com/jfrog/jfrog-cli-core/plugins/components"
)

func main() {
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
