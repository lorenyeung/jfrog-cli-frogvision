package helpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jfrog/jfrog-cli-core/utils/config"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	log "github.com/sirupsen/logrus"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"

	logFile "github.com/sirupsen/logrus"
)

//TraceData trace data struct
type TraceData struct {
	File string
	Line int
	Fn   string
}

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

func GetConfig() (*config.ArtifactoryDetails, error) {
	//TODO handle custom server id input
	serversIds, serverIdDefault, _ := GetServersIdAndDefault()
	if len(serversIds) == 0 {
		return nil, errorutils.CheckError(errors.New("no Artifactory servers configured. Use the 'jfrog rt c' command to set the Artifactory server details"))
	}

	//TODO handle if user is not admin

	//fmt.Print(serversIds, serverIdDefault)
	config, _ := config.GetArtifactorySpecificConfig(serverIdDefault, true, false)

	ping, _, _ := GetRestAPI("GET", true, config.Url+"api/system/ping", config.User, config.Password, "", nil, 1)
	if string(ping) != "OK" {
		logFile.Error("Artifactory is not up")
		return nil, errors.New("Artifactory is not up")
	}

	return config, nil
}

func GetMetricsDataRaw(config *config.ArtifactoryDetails) []byte {
	metrics, _, _ := GetRestAPI("GET", true, config.Url+"api/v1/metrics", config.User, config.Password, "", nil, 1)
	return metrics
}

func GetMetricsDataJSON(config *config.ArtifactoryDetails, prettyPrint bool) ([]byte, error) {
	metrics := GetMetricsDataRaw(config)

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

	var jsonText []byte
	var err error
	//pretty print
	if prettyPrint {
		jsonText, err := json.MarshalIndent(result, "", "    ")
		if err != nil {
			return nil, err
		}
		fmt.Println(string(jsonText))
		return jsonText, nil
	}
	jsonText, err = json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return jsonText, nil
}

func GetMetricsData(config *config.ArtifactoryDetails, counter int, prettyPrint bool) ([]Data, string, int, error) {
	//log.Info("hello")
	//TODO check if token vs password apikey
	jsonText, err := GetMetricsDataJSON(config, prettyPrint)
	if err != nil {
		return nil, "", 0, err
	}

	var metricsData []Data
	err2 := json.Unmarshal(jsonText, &metricsData)
	if err2 != nil {
		return nil, "", 0, err2
	}

	currentTime := time.Now()

	if len(metricsData) == 0 {
		counter = counter + 1
		currentTime = currentTime.Add(time.Second * -1 * time.Duration(counter))
	} else {
		counter = 0
	}
	return metricsData, currentTime.Format("2006.01.02 15:04:05"), counter, nil
}

func GetServersIdAndDefault() ([]string, string, error) {
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

func SetLogger(logLevelVar string) {
	level, err := log.ParseLevel(logLevelVar)
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)

	log.SetReportCaller(true)
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.QuoteEmptyFields = true
	customFormatter.FullTimestamp = true
	customFormatter.CallerPrettyfier = func(f *runtime.Frame) (string, string) {
		repopath := strings.Split(f.File, "/")
		//function := strings.Replace(f.Function, "go-pkgdl/", "", -1)
		return fmt.Sprintf("%s\t", f.Function), fmt.Sprintf(" %s:%d\t", repopath[len(repopath)-1], f.Line)
	}

	log.SetFormatter(customFormatter)
	fmt.Println("Log level set at ", level)
}

//Check logger for errors
func Check(e error, panicCheck bool, logs string, trace TraceData) {
	if e != nil && panicCheck {
		log.Error(logs, " failed with error:", e, " ", trace.Fn, " on line:", trace.Line)
		panic(e)
	}
	if e != nil && !panicCheck {
		log.Warn(logs, " failed with error:", e, " ", trace.Fn, " on line:", trace.Line)
	}
}

//Trace get function data
func Trace() TraceData {
	var trace TraceData
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Warn("Failed to get function data")
		return trace
	}

	fn := runtime.FuncForPC(pc)
	trace.File = file
	trace.Line = line
	trace.Fn = fn.Name()
	return trace
}

//GetRestAPI GET rest APIs response with error handling
func GetRestAPI(method string, auth bool, urlInput, userName, apiKey, providedfilepath string, header map[string]string, retry int) ([]byte, int, http.Header) {
	if retry > 5 {
		log.Warn("Exceeded retry limit, cancelling further attempts")
		return nil, 0, nil
	}

	body := new(bytes.Buffer)
	//PUT upload file
	if method == "PUT" && providedfilepath != "" {
		//req.Header.Set()
		file, err := os.Open(providedfilepath)
		Check(err, false, "open", Trace())
		defer file.Close()

		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("file", filepath.Base(providedfilepath))
		Check(err, false, "create", Trace())
		io.Copy(part, file)
		err = writer.Close()
		Check(err, false, "writer close", Trace())
	}

	client := http.Client{}
	req, err := http.NewRequest(method, urlInput, body)
	if auth {
		req.SetBasicAuth(userName, apiKey)
	}
	for x, y := range header {
		log.Debug("Recieved extra header:", x+":"+y)
		req.Header.Set(x, y)
	}

	if err != nil {
		log.Warn("The HTTP request failed with error", err)
	} else {

		resp, err := client.Do(req)
		Check(err, false, "The HTTP response", Trace())

		if err != nil {
			return nil, 0, nil
		}
		// need to account for 403s with xray, or other 403s, 429? 204 is bad too (no content for docker)
		switch resp.StatusCode {
		case 200:
			log.Debug("Received ", resp.StatusCode, " OK on ", method, " request for ", urlInput, " continuing")
		case 201:
			if method == "PUT" {
				log.Debug("Received ", resp.StatusCode, " ", method, " request for ", urlInput, " continuing")
			}
		case 403:
			log.Error("Received ", resp.StatusCode, " Forbidden on ", method, " request for ", urlInput, " continuing")
			// should we try retry here? probably not
		case 404:
			log.Debug("Received ", resp.StatusCode, " Not Found on ", method, " request for ", urlInput, " continuing")
		case 429:
			log.Error("Received ", resp.StatusCode, " Too Many Requests on ", method, " request for ", urlInput, ", sleeping then retrying, attempt ", retry)
			time.Sleep(10 * time.Second)
			GetRestAPI(method, auth, urlInput, userName, apiKey, providedfilepath, header, retry+1)
		case 204:
			if method == "GET" {
				log.Error("Received ", resp.StatusCode, " No Content on ", method, " request for ", urlInput, ", sleeping then retrying")
				time.Sleep(10 * time.Second)
				GetRestAPI(method, auth, urlInput, userName, apiKey, providedfilepath, header, retry+1)
			} else {
				log.Debug("Received ", resp.StatusCode, " OK on ", method, " request for ", urlInput, " continuing")
			}
		case 500:
			log.Error("Received ", resp.StatusCode, " Internal Server error on ", method, " request for ", urlInput, " failing out")
			return nil, 0, nil
		default:
			log.Warn("Received ", resp.StatusCode, " on ", method, " request for ", urlInput, " continuing")
		}
		//Mostly for HEAD requests
		statusCode := resp.StatusCode
		headers := resp.Header

		if providedfilepath != "" && method == "GET" {
			// Create the file
			out, err := os.Create(providedfilepath)
			Check(err, false, "File create:"+providedfilepath, Trace())
			defer out.Close()

			//done := make(chan int64)
			//go helpers.PrintDownloadPercent(done, filepath, int64(resp.ContentLength))
			_, err = io.Copy(out, resp.Body)
			Check(err, false, "The file copy:"+providedfilepath, Trace())
		} else {
			//maybe skip the download or retry if error here, like EOF
			data, err := ioutil.ReadAll(resp.Body)
			Check(err, false, "Data read:"+urlInput, Trace())
			if err != nil {
				log.Warn("Data Read on ", urlInput, " failed with:", err, ", sleeping then retrying, attempt:", retry)
				time.Sleep(10 * time.Second)

				GetRestAPI(method, auth, urlInput, userName, apiKey, providedfilepath, header, retry+1)
			}

			return data, statusCode, headers
		}
	}
	return nil, 0, nil
}
