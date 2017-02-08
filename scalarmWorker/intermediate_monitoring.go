package scalarmWorker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

// IntermediateMonitoring - executes progress monitor of a simulation run and stops when it gets a signal from the main thread
func (sim SimulationManager) IntermediateMonitoring(messages chan struct{}, finished chan struct{}, codeBaseDir string, experimentManagers []string, simIndex int,
	simulationDirPath string, client *http.Client, experimentID string) {

	communicationTimeout := 30 * time.Second

	em := ExperimentManager{
		HttpClient:           client,
		BaseUrls:             experimentManagers,
		CommunicationTimeout: communicationTimeout,
		Config:               sim.Config,
		ExperimentId:         experimentID}

	if _, err := os.Stat(path.Join(codeBaseDir, "progress_monitor")); err == nil {
		for {
			progressMonitorCmd := exec.Command("sh", "-c", path.Join(codeBaseDir, "progress_monitor >>_stdout.txt 2>&1"))
			progressMonitorCmd.Dir = simulationDirPath

			if err = progressMonitorCmd.Run(); err != nil {
				fmt.Println("[SiM] An error occurred during 'progress_monitor' execution.")
				fmt.Println("[SiM] Please check if 'progress_monitor' executes correctly on the selected infrastructure.")
				fmt.Printf("[Fatal error] occured during '%v' execution \n", strings.Join(progressMonitorCmd.Args, " "))
				fmt.Printf("[Fatal error] %s\n", err.Error())
				PrintStdoutLog()
				os.Exit(1)
			}

			intermediateResults := new(SimulationRunResults)

			if _, err := os.Stat("intermediate_result.json"); os.IsNotExist(err) {
				intermediateResults.Status = "error"
				intermediateResults.Reason = fmt.Sprintf("No 'intermediate_result.json' file found: %s", err.Error())
			} else {
				file, err := os.Open("intermediate_result.json")

				if err != nil {
					intermediateResults.Status = "error"
					intermediateResults.Reason = fmt.Sprintf("Could not open 'intermediate_result.json': %s", err.Error())
				} else {
					err = json.NewDecoder(file).Decode(&intermediateResults)

					if err != nil {
						intermediateResults.Status = "error"
						intermediateResults.Reason = fmt.Sprintf("Error during 'intermediate_result.json' parsing: %s", err.Error())
					}
				}

				file.Close()
			}

			if intermediateResults.Status == "ok" {
				data := url.Values{}
				data.Set("status", intermediateResults.Status)
				data.Add("reason", intermediateResults.Reason)
				b, _ := json.Marshal(intermediateResults.Results)
				data.Add("result", string(b))

				fmt.Printf("[SiM][progress_info] Results: %v\n", data)

				err = em.PostProgressInfo(simIndex, data)

				if err != nil {
					Fatal(err)
				}
			}

			time.Sleep(10 * time.Second)
			select {
			case _ = <-messages:
				fmt.Printf("[SiM][progress_info] Our work is finished\n")
				finished <- struct{}{}
				return
			default:
			}
		}
	} else {
		fmt.Printf("[SiM][progress_info] There is no progress monitor script\n")
		finished <- struct{}{}
	}
}
