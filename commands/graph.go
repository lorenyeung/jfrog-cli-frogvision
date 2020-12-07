package commands

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/jfrog/jfrog-cli-core/plugins/components"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"

	helpers "github.com/jfrog/frogvision/utils"

	"github.com/jfrog/jfrog-cli-core/utils/config"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func GetGraphCommand() components.Command {
	return components.Command{
		Name:        "graph",
		Description: "Graph open metrics API.",
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
	return []components.Argument{}
}

func getGraphFlags() []components.Flag {
	return []components.Flag{
		components.StringFlag{
			Name:         "interval",
			Description:  "Polling interval in seconds",
			DefaultValue: "1",
		},
	}
}

func getGraphEnvVar() []components.EnvVar {
	return []components.EnvVar{}
}

type GraphConfiguration struct {
	interval int
}

func GraphCmd(c *components.Context) error {

	interval, err := strconv.Atoi(c.GetStringFlagValue("interval"))

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
	g2.SetRect(0, 12, 51, 15)
	g2.Percent = 0
	g2.BarColor = ui.ColorGreen
	g2.LabelStyle = ui.NewStyle(ui.ColorBlue)
	g2.BorderStyle.Fg = ui.ColorWhite

	g3 := widgets.NewGauge()
	g3.Title = "Current Used Heap"
	g3.SetRect(0, 16, 51, 19)
	g3.Percent = 0
	g3.BarColor = ui.ColorGreen
	g3.LabelStyle = ui.NewStyle(ui.ColorBlue)
	g3.BorderStyle.Fg = ui.ColorWhite

	//DB connections
	g4 := widgets.NewGauge()
	g4.Title = "Active DB connections"
	g4.SetRect(0, 20, 51, 23)
	g4.Percent = 0
	g4.BarColor = ui.ColorGreen
	g4.LabelStyle = ui.NewStyle(ui.ColorBlue)
	g4.BorderStyle.Fg = ui.ColorWhite

	//plot
	p1 := widgets.NewPlot()
	p1.Title = "DB Connection Chart"
	p1.Marker = widgets.MarkerDot
	p1.Data = [][]float64{[]float64{1, 2, 3, 4, 5}}
	p1.SetRect(52, 12, 77, 23)
	p1.DotMarkerRune = '+'
	p1.AxesColor = ui.ColorWhite
	p1.LineColors[0] = ui.ColorYellow
	p1.DrawDirection = widgets.DrawLeft

	//bar chart
	barchartData := []float64{1, 1, 1, 1}

	bc := widgets.NewBarChart()
	bc.Title = "DB Connections"
	bc.BarWidth = 5
	bc.Data = barchartData
	bc.SetRect(0, 24, 77, 34)
	bc.Labels = []string{"Active", "Max", "Idle", "MinIdle"}
	bc.BarColors[0] = ui.ColorGreen
	bc.NumStyles[0] = ui.NewStyle(ui.ColorBlack)

	ui.Render(bc, g2, g3, g4, o, p, p1, q, r)

	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Second * time.Duration(interval)).C
	offSetCounter := 0
	tickerCount := 1
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
			offSetCounter, err = drawFunction(config, bc, barchartData, g2, g3, g4, o, p, p1, q, r, offSetCounter, tickerCount, interval)
			if err != nil {
				return errorutils.CheckError(err)
			}
			tickerCount++

		}
	}
}

func drawFunction(config *config.ArtifactoryDetails, bc *widgets.BarChart, bcData []float64, g2 *widgets.Gauge, g3 *widgets.Gauge, g4 *widgets.Gauge, o *widgets.Paragraph, p *widgets.Paragraph, p1 *widgets.Plot, q *widgets.Paragraph, r *widgets.Paragraph, offSetCounter int, ticker int, interval int) (int, error) {
	responseTime := time.Now()
	data, lastUpdate, offset, err := helpers.GetMetricsData(config, offSetCounter, false, interval)
	if err != nil {
		return 0, err
	}

	var freeSpace, totalSpace, heapFreeSpace, heapMaxSpace, heapTotalSpace *big.Float = big.NewFloat(1), big.NewFloat(100), big.NewFloat(100), big.NewFloat(100), big.NewFloat(100)
	var heapProc string
	var dbConnIdle, dbConnMinIdle, dbConnActive, dbConnMaxActive string
	fmt.Println(dbConnIdle, dbConnMinIdle)
	//var freeInt, totalInt int
	//maybe we can turn this into a hashtable for faster lookup
	for i := range data {

		var err error
		//TODO need logic to get more than 1 if there are multiple remote - there is a bug that halts the whole thing

		switch dataArg := data[i].Name; dataArg {
		case "sys_cpu_totaltime_seconds":
			q.Text = data[i].Metric[0].Value
		case "jfrt_runtime_heap_maxmemory_bytes":
			heapMaxSpace, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				//prevent cannot divide by zero error for all heap/space floats to prevent remote connection crashes
				heapMaxSpace = big.NewFloat(1)
				//return 0, errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(helpers.Trace().Line))
			}
		case "jfrt_runtime_heap_freememory_bytes":
			heapFreeSpace, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				heapFreeSpace = big.NewFloat(1)
				//return 0, errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(helpers.Trace().Line))
			}
		case "jfrt_runtime_heap_totalmemory_bytes":
			heapTotalSpace, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				heapTotalSpace = big.NewFloat(1)
				//return 0, errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(helpers.Trace().Line))
			}
		case "jfrt_runtime_heap_processors_total":
			heapProc = data[i].Metric[0].Value
		case "app_disk_free_bytes":
			freeSpace, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				freeSpace = big.NewFloat(1)
				//return 0, errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(helpers.Trace().Line))
			}
		case "app_disk_total_bytes":
			totalSpace, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				totalSpace = big.NewFloat(1)
				//return 0, errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(helpers.Trace().Line))
			}
		case "jfrt_db_connections_active_total":
			dbConnActive = data[i].Metric[0].Value
		case "jfrt_db_connections_max_active_total":
			dbConnMaxActive = data[i].Metric[0].Value
		case "jfrt_db_connections_min_idle_total":
			dbConnMinIdle = data[i].Metric[0].Value
		case "jfrt_db_connections_idle_total":
			dbConnIdle = data[i].Metric[0].Value
		default:
			// freebsd, openbsd,
			// plan9, windows...
			//fmt.Printf("%s.\n", os)
		}
		//repo specific connection check
		if strings.Contains(data[i].Name, "jfrt_http_connections_max_total") {
			p.Text = data[i].Metric[0].Value + " " + data[i].Metric[0].Labels.Pool + " " + data[i].Name
			//jfrt_http_connections_available_total{max
			//jfrt_http_connections_leased_total{max="50"
			//jfrt_http_connections_pending_total{max="50",
		}
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

	//compute DB active guage
	dbConnActiveInt, err := strconv.Atoi(dbConnActive)
	if err != nil {
		dbConnActiveInt = 0
		//return 0, errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(helpers.Trace().Line))
	}
	dbConnMaxActiveInt, err := strconv.Atoi(dbConnMaxActive)
	if err != nil {
		//prevent integer divide by zero error
		dbConnMaxActiveInt = 1
		//return 0, errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(helpers.Trace().Line))
	}
	dbConnIdleInt, err := strconv.Atoi(dbConnIdle)
	if err != nil {
		dbConnIdleInt = 0
		//return 0, errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(helpers.Trace().Line))
	}
	dbConnMinIdleInt, err := strconv.Atoi(dbConnMinIdle)
	if err != nil {
		dbConnMinIdleInt = 0
		//return 0, errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(helpers.Trace().Line))
	}
	pctDbConnActive := dbConnActiveInt / dbConnMaxActiveInt * 100
	g4.Percent = pctDbConnActive

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

	bc.Data = []float64{float64(dbConnActiveInt), float64(dbConnMaxActiveInt), float64(dbConnIdleInt), float64(dbConnMinIdleInt)}

	//metrics data
	r.Text = "Count: " + strconv.Itoa(len(data)) + "\nHeap Proc: " + heapProc + "\nHeap Total: " + heapTotalSpace.String()

	o.Text = "Current time: " + time.Now().Format("2006.01.02 15:04:05") + "\nLast updated: " + lastUpdate + " (" + strconv.Itoa(offset) + " seconds)\nResponse time: " + time.Now().Sub(responseTime).String()

	ui.Render(bc, g2, g3, g4, o, p, p1, q, r)
	return offset, nil
}
