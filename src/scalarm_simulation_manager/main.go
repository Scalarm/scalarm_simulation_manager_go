package main

// TODO unzipping should be cross-platform
// TODO CPU type and MHz monitoring
// TODO getting random experiment id when there is no one in the config

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	// "runtime"
	"math/rand"
	"strings"
	"time"
)

// Config file description - this should be provided by Experiment Manager in 'config.json'
type SimulationManagerConfig struct {
	ExperimentId          string `json:"experiment_id"`
	InformationServiceUrl string `json:"information_service_url"`
	ExperimentManagerUser string `json:"experiment_manager_user"`
	ExperimentManagerPass string `json:"experiment_manager_pass"`
	Development           bool   `json:"development"`
	StartAt               string `json:"start_at"`
	Timeout               int    `json:"timeout"`
}

// Results structure - we send this back to Experiment Manager
type SimulationRunResults struct {
	Status  string      `json:"status"`
	Results interface{} `json:"results"`
	Reason  string      `json:"reason"`
}

type RequestInfo struct {
	HttpMethod    string
	Body          io.Reader
	ContentType   string
	ServiceMethod string
}

func ExecuteScalarmRequest(reqInfo RequestInfo, serviceUrls []string, config *SimulationManagerConfig,
	client *http.Client, timeout time.Duration) []byte {

	protocol := "https"
	if config.Development {
		protocol = "http"
	}

	// 1. shuffle service url
	perm := rand.Perm(len(serviceUrls))

	for _, v := range perm {
		// 2. get next service url and prepare a request
		serviceUrl := serviceUrls[v]
		fmt.Printf("%s://%s/%s\n", protocol, serviceUrl, reqInfo.ServiceMethod)
		req, err := http.NewRequest(reqInfo.HttpMethod, fmt.Sprintf("%s://%s/%s", protocol, serviceUrl, reqInfo.ServiceMethod), reqInfo.Body)
		if err != nil {
			panic(err)
		}
		req.SetBasicAuth(config.ExperimentManagerUser, config.ExperimentManagerPass)
		if reqInfo.Body != nil {
			req.Header.Set("Content-Type", reqInfo.ContentType)
		}
		// 3. execute request with timeout
		response, err := GetWithTimeout(client, req, timeout)
		// 4. if response body is nil go to 2.
		if err == nil {
			return response
		}
	}

	panic("Could not execute request against Scalarm service")
}

// Calling Get multiple time until valid response or exceed 'communicationTimeout' period
func GetWithTimeout(client *http.Client, request *http.Request, communicationTimeout time.Duration) ([]byte, error) {
	var resp *http.Response
	var err error
	communicationFailed := true
	communicationStart := time.Now()
	var body []byte

	for communicationStart.Add(communicationTimeout).After(time.Now()) {
		resp, err = client.Do(request)

		if err != nil {
			time.Sleep(1 * time.Second)
			fmt.Printf("[SiM] %v\n", err)
		} else {
			communicationFailed = false
			break
		}
	}

	if communicationFailed {
		return nil, err
	}

	defer resp.Body.Close()

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return nil, err
	}

	return body, nil
}

// this method executes progress monitor of a simulation run and stops when it gets a signal from the main thread
func IntermediateMonitoring(messages chan string, codeBaseDir string, experimentManagers []string, simIndex float64, config *SimulationManagerConfig, simulationDirPath string) {
	communicationTimeout := 30 * time.Second

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	if _, err := os.Stat(path.Join(codeBaseDir, "progress_monitor")); err == nil {
		for {
			progressMonitorCmd := exec.Command("sh", "-c", path.Join(codeBaseDir, "progress_monitor >>_stdout.txt 2>&1"))
			progressMonitorCmd.Dir = simulationDirPath

			if err = progressMonitorCmd.Run(); err != nil {
				fmt.Printf("[SiM][progress_info] %v\n", err)
				break
			}

			intermediateResults := new(SimulationRunResults)

			if _, err := os.Stat("output.json"); os.IsNotExist(err) {
				intermediateResults.Status = "error"
				intermediateResults.Reason = "No 'intermediate_result.json' file found"
			} else {
				file, err := os.Open("output.json")

				if err != nil {
					intermediateResults.Status = "error"
					intermediateResults.Reason = "Could not open 'intermediate_result.json'"
				} else {
					err = json.NewDecoder(file).Decode(&intermediateResults)

					if err != nil {
						intermediateResults.Status = "error"
						intermediateResults.Reason = "Error during 'intermediate_result.json' parsing"
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

				progressInfo := RequestInfo{"POST", strings.NewReader(data.Encode()),
					"application/x-www-form-urlencoded",
					fmt.Sprintf("experiments/%v/simulations/%v/progress_info", config.ExperimentId, simIndex)}

				body := ExecuteScalarmRequest(progressInfo, experimentManagers, config, client, communicationTimeout)

				fmt.Printf("[SiM][progress_info] Response body: %s\n", body)
			}

			select {
			case _ = <-messages:
				fmt.Printf("[SiM][progress_info] Our work is finished\n")
				return
			default:
				time.Sleep(10 * time.Second)
			}
		}
	} else {
		fmt.Printf("[SiM][progress_monitor] There is no progress monitor script\n")
		<- messages
	}
}

func main() {
	var file *os.File
	var experimentDir string

	rand.Seed(time.Now().UTC().UnixNano())

	// 0. remember current location
	rootDirPath, _ := os.Getwd()
	rootDir, err := os.Open(rootDirPath)
	if err != nil {
		panic(err)
	}

	fmt.Printf("[SiM] working directory: %s\n", rootDirPath)

	// 1. load config file
	configFile, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}

	config := new(SimulationManagerConfig)
	err = json.NewDecoder(configFile).Decode(&config)
	configFile.Close()

	if err != nil {
		panic(err)
	}

	if config.Timeout <= 0 {
		config.Timeout = 60
	}
	communicationTimeout := time.Duration(config.Timeout) * time.Second

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	if len(config.StartAt) > 0 {
		startTime, err := time.Parse(time.RFC3339, config.StartAt)
		if err != nil {
			fmt.Printf("[SiM] %v\n", err)
		} else {
			fmt.Printf("We have start_at provided\n")
			for startTime.After(time.Now()) {
				time.Sleep(1 * time.Second)
			}
			fmt.Printf("We are ready to work\n")
		}
	}

	//2. getting experiment and storage manager addresses
	iSReqInfo := RequestInfo{"GET", nil, "", "experiment_managers"}
	body := ExecuteScalarmRequest(iSReqInfo, []string{config.InformationServiceUrl}, config, client, communicationTimeout)

	var experimentManagers []string

	fmt.Printf("Response body: %s.\n", body)

	if err := json.Unmarshal(body, &experimentManagers); err != nil {
		panic(err)
	}

	if len(experimentManagers) == 0 {
		panic("There is no experiment manager registered.")
	}

	// getting storage manager address
	iSReqInfo = RequestInfo{"GET", nil, "", "storage_managers"}
	body = ExecuteScalarmRequest(iSReqInfo, []string{config.InformationServiceUrl}, config, client, communicationTimeout)

	var storageManagers []string

	fmt.Printf("Response body: %s.\n", body)

	if err := json.Unmarshal(body, &storageManagers); err != nil {
		panic(err)
	}

	if len(storageManagers) == 0 {
		panic("There is no storage manager registered.")
	}

	// creating directory for experiment data
	experimentDir = path.Join(rootDirPath, fmt.Sprintf("experiment_%s", config.ExperimentId))

	if err = os.MkdirAll(experimentDir, 0777); err != nil {
		panic(err)
	}

	// 3. get code base for the experiment if necessary
	codeBaseDir := path.Join(experimentDir, "code_base")

	if _, err := os.Stat(codeBaseDir); os.IsNotExist(err) {
		if err = os.MkdirAll(codeBaseDir, 0777); err != nil {
			panic(err)
		}
		fmt.Println("Getting code base ...")
		codeBaseUrl := fmt.Sprintf("experiments/%s/code_base", config.ExperimentId)
		codeBaseInfo := RequestInfo{"GET", nil, "", codeBaseUrl}
		body = ExecuteScalarmRequest(codeBaseInfo, experimentManagers, config, client, communicationTimeout)

		w, err := os.Create(path.Join(codeBaseDir, "code_base.zip"))
		if err != nil {
			panic(err)
		}
		defer w.Close()

		if _, err = io.Copy(w, bytes.NewReader(body)); err != nil {
			panic(err)
		}

		unzipCmd := fmt.Sprintf("unzip -d \"%s\" \"%s/code_base.zip\"; unzip -d \"%s\" \"%s/simulation_binaries.zip\"", codeBaseDir, codeBaseDir, codeBaseDir, codeBaseDir)
		if err = exec.Command("sh", "-c", unzipCmd).Run(); err != nil {
			panic(err)
		}

		if err = exec.Command("sh", "-c", fmt.Sprintf("chmod a+x \"%s\"/*", codeBaseDir)).Run(); err != nil {
			panic(err)
		}
	}

	// 4. main loop for getting simulation runs of an experiment
	for {
		nextSimulationFailed := true
		communicationStart := time.Now()

		var nextSimulationBody []byte
		var simulation_run map[string]interface{}

		// 4.a getting input values for next simulation run
		for communicationStart.Add(time.Duration(int(communicationTimeout) * len(experimentManagers))).After(time.Now()) {
			fmt.Println("Getting next simulation run ...")
			nextSimulationUrl := fmt.Sprintf("experiments/%s/next_simulation", config.ExperimentId)
			nextSimulationInfo := RequestInfo{"GET", nil, "", nextSimulationUrl}
			nextSimulationBody = ExecuteScalarmRequest(nextSimulationInfo, experimentManagers, config, client, communicationTimeout)

			fmt.Printf("Next simulation: %s\n", nextSimulationBody)

			if err = json.Unmarshal(nextSimulationBody, &simulation_run); err != nil {
				fmt.Printf("[SiM] %v\n", err)
			} else {
				status := simulation_run["status"].(string)

				if status == "all_sent" {
					fmt.Printf("There is no more simulations to run in this experiment.\n")
				} else if status == "error" {
					fmt.Printf("An error occurred while getting next simulation.\n")
				} else if status != "ok" {
					fmt.Printf("We cannot continue due to unsupported status.\n")
				} else {
					nextSimulationFailed = false
					break
				}
			}

			fmt.Printf("[SiM] There was a problem while getting next simulation to run.\n")
			time.Sleep(5 * time.Second)
		}

		if nextSimulationFailed {
			panic(err)
		}

		simulation_index := simulation_run["simulation_id"].(float64)

		fmt.Printf("Simulation index: %v\n", simulation_index)
		fmt.Printf("Simulation execution constraints: %v\n", simulation_run["execution_constraints"])

		simulationDirPath := path.Join(experimentDir, fmt.Sprintf("simulation_%v", simulation_index))

		err = os.MkdirAll(simulationDirPath, 0777)
		if err != nil {
			panic(err)
		}

		input_parameters, _ := json.Marshal(simulation_run["input_parameters"].(map[string]interface{}))

		err = ioutil.WriteFile(path.Join(simulationDirPath, "input.json"), input_parameters, 0777)
		if err != nil {
			panic(err)
		}

		simulationDir, err := os.Open(simulationDirPath)
		if err != nil {
			panic(err)
		}

		wd, err := os.Getwd()
		fmt.Printf("Working dir: %v\n", wd)
		if err = simulationDir.Chdir(); err != nil {
			panic(err)
		}
		wd, err = os.Getwd()

		// 4b. run an adapter script (input writer) for input information: input.json -> some specific code
		if _, err := os.Stat(path.Join(codeBaseDir, "input_writer")); err == nil {
			fmt.Println("Before input writer ...")
			inputWriterCmd := exec.Command("sh", "-c", path.Join(codeBaseDir, "input_writer input.json >>_stdout.txt 2>&1"))
			inputWriterCmd.Dir = simulationDirPath
			if err = inputWriterCmd.Run(); err != nil {
				panic(err)
			}
			fmt.Println("After input writer ...")
		}

		// 4c.1. progress monitoring scheduling if available - TODO
		messages := make(chan string, 10)
		go IntermediateMonitoring(messages, codeBaseDir, experimentManagers, simulation_index, config, simulationDirPath)

		// 4c. run an executor of this simulation
		fmt.Println("Before executor ...")
		executorCmd := exec.Command("sh", "-c", path.Join(codeBaseDir, "executor >>_stdout.txt 2>&1"))
		executorCmd.Dir = simulationDirPath
		if err = executorCmd.Run(); err != nil {
			panic(err)
		}
		fmt.Println("After executor ...")

		messages <- "done"
		close(messages)

		// 4d. run an adapter script (output reader) to transform specific output format to scalarm model (output.json)
		if _, err := os.Stat(path.Join(codeBaseDir, "output_reader")); err == nil {
			fmt.Println("Before output reader ...")
			outputReaderCmd := exec.Command("sh", "-c", path.Join(codeBaseDir, "output_reader >>_stdout.txt 2>&1"))
			outputReaderCmd.Dir = simulationDirPath
			if err = outputReaderCmd.Run(); err != nil {
				panic(err)
			}
			fmt.Println("After output reader ...")
		}

		// 4e. upload output json to experiment manager and set the run simulation as done
		simulationRunResults := new(SimulationRunResults)

		if _, err := os.Stat("output.json"); os.IsNotExist(err) {
			simulationRunResults.Status = "error"
			simulationRunResults.Reason = "No output.json file found"
		} else {
			file, err = os.Open("output.json")

			if err != nil {
				simulationRunResults.Status = "error"
				simulationRunResults.Reason = "Could not open output.json"
			} else {
				err = json.NewDecoder(file).Decode(&simulationRunResults)

				if err != nil {
					simulationRunResults.Status = "error"
					simulationRunResults.Reason = "Error during output.json parsing"
				}
			}

			file.Close()
		}

		// 4f. upload structural results of a simulation run
		data := url.Values{}
		data.Set("status", simulationRunResults.Status)
		data.Add("reason", simulationRunResults.Reason)
		b, _ := json.Marshal(simulationRunResults.Results)
		data.Add("result", string(b))

		fmt.Printf("Results: %v\n", data)

		markAsCompleteUrl := fmt.Sprintf("experiments/%s/simulations/%v/mark_as_complete", config.ExperimentId, simulation_index)
		markAsCompleteInfo := RequestInfo{"POST", strings.NewReader(data.Encode()), "application/x-www-form-urlencoded",
			markAsCompleteUrl}
		body = ExecuteScalarmRequest(markAsCompleteInfo, experimentManagers, config, client, communicationTimeout)

		fmt.Printf("Response body: %s\n", body)

		if len(storageManagers) > 0 {
			// 4g. upload binary output if provided
			if _, err := os.Stat("output.tar.gz"); err == nil {
				fmt.Printf("Uploading output.tar.gz ...\n")
				file, err := os.Open("output.tar.gz")

				if err != nil {
					panic(err)
				}

				defer file.Close()

				requestBody := &bytes.Buffer{}
				writer := multipart.NewWriter(requestBody)
				part, err := writer.CreateFormFile("file", filepath.Base("output.tar.gz"))
				if err != nil {
					panic(err)
				}
				_, err = io.Copy(part, file)

				err = writer.Close()
				if err != nil {
					panic(err)
				}

				binariesUploadUrl := fmt.Sprintf("experiments/%s/simulations/%v", config.ExperimentId, simulation_index)
				binariesUploadUrlInfo := RequestInfo{"PUT", requestBody, writer.FormDataContentType(), binariesUploadUrl}
				body = ExecuteScalarmRequest(binariesUploadUrlInfo, storageManagers, config, client, communicationTimeout)

				fmt.Printf("Response body: %s\n", body)
			}

			// 4h. upload stdout if provided
			if _, err := os.Stat("_stdout.txt"); err == nil {
				fmt.Printf("[SiM] uploading stdout...\n")

				file, err := os.Open("_stdout.txt")
				if err != nil {
					panic(err)
				}

				requestBody := &bytes.Buffer{}
				writer := multipart.NewWriter(requestBody)
				part, err := writer.CreateFormFile("file", filepath.Base("_stdout.txt"))
				if err != nil {
					panic(err)
				}
				_, err = io.Copy(part, file)
				file.Close()

				err = writer.Close()
				if err != nil {
					panic(err)
				}

				stdoutUploadUrl := fmt.Sprintf("experiments/%s/simulations/%v/stdout", config.ExperimentId, simulation_index)
				stdoutUploadUrlInfo := RequestInfo{"PUT", requestBody, writer.FormDataContentType(), stdoutUploadUrl}
				body = ExecuteScalarmRequest(stdoutUploadUrlInfo, storageManagers, config, client, communicationTimeout)

				fmt.Printf("Response body: %s\n", body)
			}
		}

		// 5. clean up - removing simulation dir
		os.RemoveAll(simulationDirPath)

		// 6. going to the root dir and moving
		if err = rootDir.Chdir(); err != nil {
			panic(err)
		}
	}
}
