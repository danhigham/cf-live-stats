package main

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry/cli/cf/configuration/config_helpers"
	"github.com/cloudfoundry/cli/cf/configuration/core_config"
	"github.com/cloudfoundry/cli/plugin"
	"github.com/mitchellh/colorstring"
)

const DATE_LAYOUT = "2006-01-02 15:04:05 -0700"
const CHART_SPAN = 300
const UPDATE_INTERVAL = 3

var UsageHistory []GroupedStat

// *** App Search Results *** //

type AppSearchResults struct {
	Resources []AppSearchResoures `json:"resources"`
}

type AppSearchResoures struct {
	Metadata AppSearchMetaData `json:"metadata"`
}

type AppSearchMetaData struct {
	Guid string `json:"guid"`
	Url  string `json:"url"`
}

// *** App Stats *** //

type AppStat struct {
	State         string        `json:"state"`
	InstanceStats InstanceStats `json:"stats"`
}

type InstanceStats struct {
	Name  string        `json:"name"`
	Usage InstanceUsage `json:"usage"`
}

type InstanceUsage struct {
	// "time":"2014-12-07 16:41:05 +0000","cpu":0.0006444376275381359,"mem":62304256,"disk":59023360
	Time string  `json:"time"`
	Cpu  float64 `json:"cpu"`
	Mem  uint64  `json:"mem"`
	Disk uint64  `json:"disk"`
}

func fatalIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stdout, "error:", err)
		os.Exit(1)
	}
}

func main() {
	plugin.Start(&InfoPlugin{})
}

type InfoPlugin struct {
	UsageHistory []GroupedStat
	Instances    int
}

type GroupedStat struct {
	TimeStamp time.Time
	Usage     map[string]AppStat
}

func (plugin InfoPlugin) HttpHandler(w http.ResponseWriter, r *http.Request) {

	path := r.URL.Path[1:]
	if path == "" {
		path = "index.html"
	}

	asset, _ := Asset(fmt.Sprintf("assets/%v", path))
	ctype := mime.TypeByExtension(filepath.Ext(path))

	w.Header().Set("Content-Type", ctype)
	fmt.Fprint(w, string(asset))
}

func (plugin InfoPlugin) Run(cliConnection plugin.CliConnection, args []string) {

	// find app guid
	appName := args[1]
	httpPort := "8080"

	if len(args) > 2 {
		httpPort = args[2]
	}

	guid := plugin.FindAppGuid(cliConnection, appName)

	// init stats data
	plugin.Instances, plugin.UsageHistory = plugin.InitData(cliConnection, guid)

	// start separate go proc to update
	go func() {
		for {
			stats := plugin.GetAppStats(cliConnection, guid)
			timeStamp, _ := time.Parse(DATE_LAYOUT, stats["0"].InstanceStats.Usage.Time)

			usage := GroupedStat{}
			usage.TimeStamp = timeStamp
			usage.Usage = stats

			plugin.UsageHistory = append([]GroupedStat{usage}, plugin.UsageHistory...)
			plugin.UsageHistory = plugin.UsageHistory[0 : len(plugin.UsageHistory)-1]

			time.Sleep(UPDATE_INTERVAL * time.Second)
		}
	}()

	// start http server
	http.HandleFunc("/data.json", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")

		out := []map[string]string{}

		for _, record := range plugin.UsageHistory {

			flatRecord := map[string]string{}
			flatRecord["time"] = record.TimeStamp.String()

			for i, usage := range record.Usage {

				flatRecord[fmt.Sprintf("cpu_%v", i)] = strconv.FormatFloat(usage.InstanceStats.Usage.Cpu, 'f', 6, 64)
				flatRecord[fmt.Sprintf("mem_%v", i)] = strconv.FormatUint(usage.InstanceStats.Usage.Mem, 10)
				flatRecord[fmt.Sprintf("disk_%v", i)] = strconv.FormatUint(usage.InstanceStats.Usage.Disk, 10)
			}

			out = append(out, flatRecord)
		}

		jsonVal, _ := json.Marshal(out)
		fmt.Fprint(w, string(jsonVal))
	})

	http.HandleFunc("/", plugin.HttpHandler)

	fmt.Println(colorstring.Color("HTTP server listening at [blue]http://localhost:" + httpPort))
	http.ListenAndServe(fmt.Sprintf(":%v", httpPort), nil)
}

func (plugin InfoPlugin) InitData(cliConnection plugin.CliConnection, appGuid string) (int, []GroupedStat) {

	// TODO: bail if the guid is nil

	stats := plugin.GetAppStats(cliConnection, appGuid)

	history := []GroupedStat{}

	timeStamp, _ := time.Parse(DATE_LAYOUT, stats["0"].InstanceStats.Usage.Time)

	// create initial value
	usage := GroupedStat{}
	usage.TimeStamp = timeStamp
	usage.Usage = stats

	for i := 0; i < CHART_SPAN; i += UPDATE_INTERVAL {
		usage = GroupedStat{}
		interval, _ := time.ParseDuration(fmt.Sprintf("-%vs", i))
		usage.TimeStamp = timeStamp.Add(interval)
		usage.Usage = stats //TODO zero out values
		history = append(history, usage)
	}

	return len(stats), history
}

func (plugin InfoPlugin) GetAppStats(cliConnection plugin.CliConnection, appGuid string) map[string]AppStat {

	appQuery := fmt.Sprintf("/v2/apps/%v/stats", appGuid)
	cmd := []string{"curl", appQuery}

	output, _ := cliConnection.CliCommandWithoutTerminalOutput(cmd...)
	res := map[string]AppStat{}
	json.Unmarshal([]byte(strings.Join(output, "")), &res)

	return res
}

func (plugin InfoPlugin) FindAppGuid(cliConnection plugin.CliConnection, appName string) string {

	confRepo := core_config.NewRepositoryFromFilepath(config_helpers.DefaultFilePath(), fatalIf)
	spaceGuid := confRepo.SpaceFields().Guid

	appQuery := fmt.Sprintf("/v2/spaces/%v/apps?q=name:%v&inline-relations-depth=1", spaceGuid, appName)
	cmd := []string{"curl", appQuery}

	output, _ := cliConnection.CliCommandWithoutTerminalOutput(cmd...)
	res := &AppSearchResults{}
	json.Unmarshal([]byte(strings.Join(output, "")), &res)

	return res.Resources[0].Metadata.Guid
}

func (InfoPlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "Stats",
		Commands: []plugin.Command{
			{
				Name:     "stats",
				HelpText: "Show browser based stats",
			},
		},
	}
}
