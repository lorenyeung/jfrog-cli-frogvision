package commands

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/jfrog/jfrog-cli-core/plugins/components"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"

	helpers "github.com/jfrog/jfrog-cli-plugin-template/utils"

	"github.com/jfrog/jfrog-cli-core/utils/config"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
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

func GraphCmd(c *components.Context) error {

	config, err := helpers.GetConfig()
	if err != nil {
		return err
	}

	if err := ui.Init(); err != nil {
		fmt.Printf("failed to initialize termui: %v", err)
		return err
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
	g2.Title = "Current Used Storage"
	g2.SetRect(0, 12, 50, 15)
	g2.Percent = 0
	g2.BarColor = ui.ColorGreen
	g2.LabelStyle = ui.NewStyle(ui.ColorBlue)
	g2.BorderStyle.Fg = ui.ColorWhite

	g3 := widgets.NewGauge()
	g3.Title = "Current Used Heap"
	g3.SetRect(0, 16, 50, 19)
	g3.Percent = 0
	g3.BarColor = ui.ColorGreen
	g3.LabelStyle = ui.NewStyle(ui.ColorBlue)
	g3.BorderStyle.Fg = ui.ColorWhite

	ui.Render(g2, g3, o, p, q, r)

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
			offSetCounter, err = drawFunction(config, g2, g3, o, p, q, r, offSetCounter)
			if err != nil {
				return errorutils.CheckError(err)
			}

		}
	}
}

func drawFunction(config *config.ArtifactoryDetails, g2 *widgets.Gauge, g3 *widgets.Gauge, o *widgets.Paragraph, p *widgets.Paragraph, q *widgets.Paragraph, r *widgets.Paragraph, offSetCounter int) (int, error) {
	responseTime := time.Now()
	data, lastUpdate, offset, err := helpers.GetMetricsData(config, offSetCounter, false)
	if err != nil {
		return 0, err
	}

	var freeSpace, totalSpace, heapFreeSpace, heapMaxSpace, heapTotalSpace *big.Float
	var heapProc string
	//var freeInt, totalInt int
	//maybe we can turn this into a hashtable for faster lookup
	for i := range data {

		var err error
		//TODO need logic to get more than 1 if there are multiple remote - there is a bug that halts the whole thing
		if data[i].Name == "jfrt_http_connections_max_total" {
			p.Text = data[i].Metric[0].Value + " " + data[i].Metric[0].Labels.Pool
			//jfrt_http_connections_available_total{max
			//jfrt_http_connections_leased_total{max="50"
			//jfrt_http_connections_pending_total{max="50",
		}
		if data[i].Name == "sys_cpu_totaltime_seconds" {
			q.Text = data[i].Metric[0].Value
		}
		if data[i].Name == "app_disk_free_bytes" {
			freeSpace, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				return 0, errorutils.CheckError(err)
			}
		}
		if data[i].Name == "app_disk_total_bytes" {
			totalSpace, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				return 0, errorutils.CheckError(err)
			}
		}
		if data[i].Name == "jfrt_runtime_heap_processors_total" {
			heapProc = data[i].Metric[0].Value
		}
		if data[i].Name == "jfrt_runtime_heap_freememory_bytes" {
			heapFreeSpace, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				return 0, errorutils.CheckError(err)
			}
		}
		if data[i].Name == "jfrt_runtime_heap_maxmemory_bytes" {
			heapMaxSpace, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				return 0, errorutils.CheckError(err)
			}
		}
		if data[i].Name == "jfrt_runtime_heap_totalmemory_bytes" {
			heapTotalSpace, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				return 0, errorutils.CheckError(err)
			}
		}
		// jfrt_runtime_heap_freememory_bytes  (float)
		// jfrt_runtime_heap_maxmemory_bytes
		// jfrt_runtime_heap_totalmemory_bytes

		// more GC metrics to consider
		// # TYPE jfrt_artifacts_gc_duration_seconds gauge
		// jfrt_artifacts_gc_duration_seconds{end="1607284801199",start="1607284800142",status="COMPLETED",type="FULL"} 1.057 1607287853275
		// # HELP jfrt_artifacts_gc_size_cleaned_bytes Total Bytes recovered by Garbage Collection
		// # UPDATED jfrt_artifacts_gc_size_cleaned_bytes 1607284811440
		// # TYPE jfrt_artifacts_gc_size_cleaned_bytes gauge
		// jfrt_artifacts_gc_size_cleaned_bytes{end="1607284801199",start="1607284800142",status="COMPLETED",type="FULL"} 5.023346e+07 1607287853275
		// # HELP jfrt_artifacts_gc_binaries_total Total number of binaries removed by Garbage Collection
		// # UPDATED jfrt_artifacts_gc_binaries_total 1607284811440
		// # TYPE jfrt_artifacts_gc_binaries_total counter
		// jfrt_artifacts_gc_binaries_total{end="1607284801199",start="1607284800142",status="COMPLETED",type="FULL"} 21 1607287853275
		// # HELP jfrt_artifacts_gc_current_size_bytes Total space occupied by binaries after Garbage Collection
		// # UPDATED jfrt_artifacts_gc_current_size_bytes 1607284811440
		// # TYPE jfrt_artifacts_gc_current_size_bytes gauge
		// jfrt_artifacts_gc_current_size_bytes{end="1607284801199",start="1607284800142",status="COMPLETED",type="FULL"} 3.823509e+10 1607287853275

	}

	//heapMax is xmx confirmed. no idea what the other two are
	//2.07e8, 4.29e09, 1.5e09
	//fmt.Println(heapFreeSpace, heapMaxSpace, heapTotalSpace)

	//compute free space gauge
	pctFreeSpace := new(big.Float).Mul(big.NewFloat(100), new(big.Float).Quo(freeSpace, totalSpace))
	pctFreeSpaceStr := pctFreeSpace.String()
	pctFreeSplit := strings.Split(pctFreeSpaceStr, ".")
	pctFreeInt, _ := strconv.Atoi(pctFreeSplit[0])
	g2.Percent = 100 - pctFreeInt

	//compute free heap gauge
	pctFreeHeapSpace := new(big.Float).Mul(big.NewFloat(100), new(big.Float).Quo(heapFreeSpace, heapMaxSpace))
	pctFreeHeapSpaceStr := pctFreeHeapSpace.String()
	pctFreeHeapSplit := strings.Split(pctFreeHeapSpaceStr, ".")
	pctFreeHeapInt, _ := strconv.Atoi(pctFreeHeapSplit[0])
	g3.Percent = pctFreeHeapInt

	//metrics data
	r.Text = "Count: " + strconv.Itoa(len(data)) + "\nHeap Proc: " + heapProc + "\nHeap Total: " + heapTotalSpace.String()

	o.Text = "Current time: " + time.Now().Format("2006.01.02 15:04:05") + "\nLast updated: " + lastUpdate + " (" + strconv.Itoa(offset) + " seconds)\nResponse time: " + time.Now().Sub(responseTime).String()

	ui.Render(g2, g3, o, p, q, r)
	return offset, nil
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
