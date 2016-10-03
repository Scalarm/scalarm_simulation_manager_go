package main

// TODO CPU type and MHz monitoring
// TODO getting random experiment id when there is no one in the config

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"crypto/x509"
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
	"container/list"
	"flag"
	"math/rand"
	"strings"
	"time"

	scalarm_worker "github.com/scalarm/scalarm_simulation_manager_go/scalarm_worker"
)

const VERSION string = "2016.10.03-1"

// Results structure - we send this back to Experiment Manager
type SimulationRunResults struct {
	Status  string      `json:"status"`
	Results interface{} `json:"results"`
	Reason  string      `json:"reason"`
}

func (res *SimulationRunResults) isValid() bool {
	return (res.Status == "ok" && res.Results != nil) || (res.Status == "error" && res.Reason != "")
}

type RequestInfo struct {
	HttpMethod    string
	Body          io.Reader
	ContentType   string
	ServiceMethod string
}

func Fatal(err error) {
	fmt.Printf("[Fatal error] %v\n", err)
	os.Exit(1)
}

func PrintStdoutLog() {
	linesNum := "100" // TODO: make int strconv.Itoa(linesNum)
	stdoutPath := "_stdout.txt"
	out, _ := exec.Command("tail", "-n", linesNum, stdoutPath).CombinedOutput()
	fmt.Printf("----------\nLast %v lines of %v:\n----------\n", linesNum, stdoutPath)
	fmt.Println(string(out))
}

func cloneZipItem(f *zip.File, dest string) error {
	//create full directory path
	path := filepath.Join(dest, f.Name)

	err := os.MkdirAll(filepath.Dir(path), os.ModeDir|os.ModePerm)
	if err != nil {
		return err
	}

	//clone if item is a file
	rc, err := f.Open()
	if err != nil {
		return err
	}

	if !f.FileInfo().IsDir() {

		fileCopy, err := os.Create(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(fileCopy, rc)
		fileCopy.Close()
		if err != nil {
			return err
		}
	}
	rc.Close()
	return nil
}

func Extract(zip_path, dest string) error {
	r, err := zip.OpenReader(zip_path)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		err = cloneZipItem(f, dest)
		if err != nil {
			return err
		}
	}

	return nil
}

func ExecuteScalarmRequest(reqInfo RequestInfo, serviceUrls []string, config *scalarm_worker.SimulationManagerConfig,
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
		fmt.Printf("[SiM] %s://%s/%s\n", protocol, serviceUrl, reqInfo.ServiceMethod)
		req, err := http.NewRequest(reqInfo.HttpMethod, fmt.Sprintf("%s://%s/%s", protocol, serviceUrl, reqInfo.ServiceMethod), reqInfo.Body)
		if err != nil {
			Fatal(err)
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

	Fatal(fmt.Errorf("Could not execute request against Scalarm service"))
	return nil
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
func IntermediateMonitoring(messages chan struct{}, finished chan struct{}, codeBaseDir string, experimentManagers []string, simIndex int,
	config *scalarm_worker.SimulationManagerConfig, simulationDirPath string, client *http.Client, experimentId string) {

	communicationTimeout := 30 * time.Second

	em := scalarm_worker.ExperimentManager{
		HttpClient:           client,
		BaseUrls:             experimentManagers,
		CommunicationTimeout: communicationTimeout,
		Config:               config,
		ExperimentId:         experimentId}

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

// Makes request to experiments/random_experiment
// Returns String: random experiment id available for current user
func GetRandomExperimentId(config *scalarm_worker.SimulationManagerConfig, experimentManagers []string, client *http.Client) string {
	communicationTimeout := 30 * time.Second
	fmt.Printf("[SiM] Getting random experiment id...\n")
	getExpReqInfo := RequestInfo{"GET", nil, "", "experiments/random_experiment"}
	body := ExecuteScalarmRequest(getExpReqInfo, experimentManagers, config, client, communicationTimeout)
	fmt.Printf("[SiM] Random experiment response body: %s\n", body)
	return fmt.Sprintf("%s", body)
}

func listIncludeString(l *list.List, a string) bool {
	for e := l.Front(); e != nil; e = e.Next() {
		if e.Value == a {
			return true
		}
	}
	return false
}

func main() {
	fmt.Printf("[SiM] Scalarm Simulation Manager, version: %s\n", VERSION)

	var file *os.File
	var experimentDir string

	rand.Seed(time.Now().UTC().UnixNano())

	// 0. remember current location
	rootDirPath, _ := os.Getwd()
	rootDir, err := os.Open(rootDirPath)
	if err != nil {
		Fatal(err)
	}

	fmt.Printf("[SiM] working directory: %s\n", rootDirPath)

	// 1. load config file
	config, err := scalarm_worker.CreateSimulationManagerConfig("config.json")
	if err != nil {
		Fatal(err)
	}

	// TODO: use flags with options to override all config values
	simulationsLimitPtr := flag.Int("simulations_limit", -1, "max number of simulation run to execute")
	flag.Parse()

	simulationsLimit := *simulationsLimitPtr

	// use config json if simulations limit not provided in command line
	if simulationsLimit == -1 {
		simulationsLimit = config.SimulationsLimit
	}

	if simulationsLimit > 0 {
		fmt.Printf("[SiM] Simulations limit set to %v\n", simulationsLimit)
	}

	if config.Timeout <= 0 {
		config.Timeout = 60
	}
	communicationTimeout := time.Duration(config.Timeout) * time.Second

	// -- HTTP client --

	var client *http.Client
	tlsConfig := tls.Config{InsecureSkipVerify: config.InsecureSSL}

	if config.ScalarmCertificatePath != "" {
		CA_Pool := x509.NewCertPool()
		severCert, err := ioutil.ReadFile(config.ScalarmCertificatePath)
		if err != nil {
			Fatal(fmt.Errorf("Could not load Scalarm certificate"))
		}
		CA_Pool.AppendCertsFromPEM(severCert)

		tlsConfig.RootCAs = CA_Pool
	}

	client = &http.Client{Transport: &http.Transport{TLSClientConfig: &tlsConfig}}

	// --

	if len(config.StartAt) > 0 {
		startTime, err := time.Parse(time.RFC3339, config.StartAt)
		if err != nil {
			fmt.Printf("[SiM] %v\n", err)
		} else {
			fmt.Println("[SiM] We have start_at provided")
			time.Sleep(startTime.Sub(time.Now()))
			fmt.Println("[SiM] We are ready to work")
		}
	}

	//2. getting experiment and storage manager addresses
	is := scalarm_worker.InformationService{
		HttpClient:           client,
		BaseUrl:              config.InformationServiceUrl,
		CommunicationTimeout: communicationTimeout,
		Config:               config}

	var experimentManagers []string
	experimentManagers, err = is.GetExperimentManagers()
	if err != nil {
		Fatal(err)
	}

	// getting storage manager address
	var storageManagers []string
	storageManagers, err = is.GetStorageManagers()
	if err != nil {
		Fatal(err)
	}

	var experimentId string
	executedExperiments := list.New()
	singleExperiment := false
	// a great loop for multiple experiments
	for {
		// get experiment_id from EM if not present in SiM config
		if config.ExperimentId == "" {
			experimentId = ""
			for experimentId == "" {
				experimentId = GetRandomExperimentId(config, experimentManagers, client)

				if experimentId == "" {
					fmt.Printf("[SiM] Random experiment id empty, waiting 30 seconds to try again\n")
					time.Sleep(30 * time.Second)

					// check if this experiment was executed by this SiM
				} else if listIncludeString(executedExperiments, experimentId) {
					fmt.Printf("[SiM] That experiment was already executed, waiting 10 seconds to get other id\n")
					experimentId = ""
					time.Sleep(10 * time.Second)

					// its new experiment - add it to executed list
				} else {
					executedExperiments.PushBack(experimentId)
				}
			}
		} else {
			experimentId = config.ExperimentId
			singleExperiment = true
		}

		// creating directory for experiment data
		experimentDir = path.Join(rootDirPath, fmt.Sprintf("experiment_%s", experimentId))

		em := scalarm_worker.ExperimentManager{
			HttpClient:           client,
			BaseUrls:             experimentManagers,
			CommunicationTimeout: communicationTimeout,
			Config:               config,
			ExperimentId:         experimentId}

		if err = os.MkdirAll(experimentDir, 0777); err != nil {
			Fatal(err)
		}

		// 3. get code base for the experiment if necessary
		codeBaseDir := path.Join(experimentDir, "code_base")

		if _, err := os.Stat(codeBaseDir); os.IsNotExist(err) {
			if err = os.MkdirAll(codeBaseDir, 0777); err != nil {
				Fatal(err)
			}

			for i := 0; i < 10; i++ {
				fmt.Println("[SiM] Getting code base ...")

				err = em.DownloadExperimentCodeBase(codeBaseDir)
				if err != nil {
					fmt.Printf("[SiM] There was a problem while getting code base: %v\n", err)
				} else {

					if err = Extract(codeBaseDir+"/code_base.zip", codeBaseDir); err != nil {
						fmt.Println("[SiM] An error occurred while unzipping 'code_base.zip'.")
						fmt.Println("[Error] occured while unzipping 'code_base.zip'.")
						fmt.Printf("[Error] %s\n", err.Error())
					}

					if err = Extract(codeBaseDir+"/simulation_binaries.zip", codeBaseDir); err != nil {
						fmt.Println("[SiM] An error occurred while unzipping 'simulation_binaries.zip'.")
						fmt.Println("[Error] occured while unzipping 'simulation_binaries.zip'.")
						fmt.Printf("[Error] %s\n", err.Error())
					}
				}

				if err == nil {
					break
				} else {
					time.Sleep(5 * time.Second)
				}
			}

			if err = exec.Command("sh", "-c", fmt.Sprintf("chmod a+x \"%s\"/*", codeBaseDir)).Run(); err != nil {
				fmt.Println("[SiM] An error occurred during executing 'chmod' command. Please check if you have required permissions.")
				fmt.Printf("[Fatal error] occured during '%v' execution \n", fmt.Sprintf("chmod a+x \"%s\"/*", codeBaseDir))
				fmt.Printf("[Fatal error] %s\n", err.Error())
				os.Exit(2)
			}
		}

		// 4. main loop for getting simulation runs of an experiment
		simulationsDone := 0
		for {
			nextSimulationFailed := true
			communicationStart := time.Now()

			var simulation_run map[string]interface{}
			wait := false

			// 4.a getting input values for next simulation run
			for communicationStart.Add(communicationTimeout * time.Duration(len(experimentManagers))).After(time.Now()) {
				fmt.Println("[SiM] Getting next simulation run ...")
				simulation_run, err = em.GetNextSimulationRunConfig()

				if err != nil {
					Fatal(err)
				}

				status := simulation_run["status"].(string)

				if status == "all_sent" {
					fmt.Println("[SiM] There is no more simulations to run in this experiment.")
				} else if status == "error" {
					fmt.Println("[SiM] An error occurred while getting next simulation.")
				} else if status == "wait" {
					fmt.Printf("[SiM] There is no more simulations to run in this experiment "+
						"at the moment, time to wait: %vs\n", simulation_run["duration_in_seconds"])
					wait = true
					break
				} else if status != "ok" {
					fmt.Println("[SiM] We cannot continue due to unsupported status.")
				} else {
					nextSimulationFailed = false
					break
				}

				fmt.Println("[SiM] There was a problem while getting next simulation to run.")
				time.Sleep(5 * time.Second)
			}
			if wait {
				time.Sleep(time.Duration(simulation_run["duration_in_seconds"].(float64)) * time.Second)
				continue
			}

			if nextSimulationFailed {
				fmt.Println("[SiM] Couldn't get simulation to run")
				if singleExperiment {
					fmt.Println("[SiM] that was single experiment run -> finishing work.")
					os.Exit(0)
				} else {
					fmt.Println("[SiM] will try another experiment")
					break
				}
			}

			simulation_index := int(simulation_run["simulation_id"].(float64))

			fmt.Printf("[SiM] Simulation index: %v\n", simulation_index)
			fmt.Printf("[SiM] Simulation execution constraints: %v\n", simulation_run["execution_constraints"])

			simulationDirPath := path.Join(experimentDir, fmt.Sprintf("simulation_%v", simulation_index))

			err = os.MkdirAll(simulationDirPath, 0777)
			if err != nil {
				Fatal(err)
			}

			input_parameters, _ := json.Marshal(simulation_run["input_parameters"].(map[string]interface{}))

			err = ioutil.WriteFile(path.Join(simulationDirPath, "input.json"), input_parameters, 0777)
			if err != nil {
				Fatal(err)
			}

			simulationDir, err := os.Open(simulationDirPath)
			if err != nil {
				Fatal(err)
			}

			wd, err := os.Getwd()
			fmt.Printf("[SiM] Working dir: %v\n", wd)
			if err = simulationDir.Chdir(); err != nil {
				Fatal(err)
			}
			wd, err = os.Getwd()

			// 4b. run an adapter script (input writer) for input information: input.json -> some specific code
			if _, err := os.Stat(path.Join(codeBaseDir, "input_writer")); err == nil {
				fmt.Println("[SiM] Before input writer ...")
				inputWriterCmd := exec.Command("sh", "-c", path.Join(codeBaseDir, "input_writer input.json >>_stdout.txt 2>&1"))
				inputWriterCmd.Dir = simulationDirPath
				if err = inputWriterCmd.Run(); err != nil {
					fmt.Println("[SiM] An error occurred during 'input_writer' execution.")
					fmt.Println("[SiM] Please check if 'input_writer' executes correctly on the selected infrastructure.")
					fmt.Printf("[Fatal error] occured during '%v' execution \n", strings.Join(inputWriterCmd.Args, " "))
					fmt.Printf("[Fatal error] %s\n", err.Error())
					PrintStdoutLog()
					os.Exit(1)
				}
				fmt.Println("[SiM] After input writer ...")
			}

			// 4c.1. progress monitoring scheduling if available - TODO
			messages := make(chan struct{}, 1)
			finished := make(chan struct{}, 1)
			go IntermediateMonitoring(messages, finished, codeBaseDir, experimentManagers, simulation_index, config, simulationDirPath, client, experimentId)

			// 4c. run an executor of this simulation
			fmt.Println("[SiM] Before executor ...")
			executorCmd := exec.Command("sh", "-c", path.Join(codeBaseDir, "executor >>_stdout.txt 2>&1"))
			executorCmd.Dir = simulationDirPath
			if err = executorCmd.Run(); err != nil {
				fmt.Println("[SiM] An error occurred during 'executor' execution.")
				fmt.Println("[SiM] Please check if 'executor' executes correctly on the selected infrastructure.")
				fmt.Printf("[Fatal error] occured during '%v' execution \n", strings.Join(executorCmd.Args, " "))
				fmt.Printf("[Fatal error] %s\n", err.Error())
				PrintStdoutLog()
				os.Exit(1)
			}
			fmt.Println("[SiM] After executor ...")

			messages <- struct{}{}
			close(messages)

			// 4d. run an adapter script (output reader) to transform specific output format to scalarm model (output.json)
			if _, err := os.Stat(path.Join(codeBaseDir, "output_reader")); err == nil {
				fmt.Println("[SiM] Before output reader ...")
				outputReaderCmd := exec.Command("sh", "-c", path.Join(codeBaseDir, "output_reader >>_stdout.txt 2>&1"))
				outputReaderCmd.Dir = simulationDirPath
				if err = outputReaderCmd.Run(); err != nil {
					fmt.Println("[SiM] An error occurred during 'output_reader' execution.")
					fmt.Println("[SiM] Please check if 'output_reader' executes correctly on the selected infrastructure.")
					fmt.Printf("[Fatal error] occured during '%v' execution \n", strings.Join(outputReaderCmd.Args, " "))
					fmt.Printf("[Fatal error] %s\n", err.Error())
					PrintStdoutLog()
					os.Exit(1)
				}
				fmt.Println("[SiM] After output reader ...")
			}

			// 4e. upload output json to experiment manager and set the run simulation as done
			simulationRunResults := new(SimulationRunResults)

			if _, err := os.Stat("output.json"); os.IsNotExist(err) {
				simulationRunResults.Status = "error"
				simulationRunResults.Reason = fmt.Sprintf("No output.json file found: %s", err.Error())
			} else {
				file, err = os.Open("output.json")

				if err != nil {
					simulationRunResults.Status = "error"
					simulationRunResults.Reason = fmt.Sprintf("Could not open output.json: %s", err.Error())
				} else {
					err = json.NewDecoder(file).Decode(&simulationRunResults)

					if err != nil {
						simulationRunResults.Status = "error"
						simulationRunResults.Reason = fmt.Sprintf("Error during output.json parsing: %s", err.Error())
					}
				}

				file.Close()
			}

			resultJson, _ := json.Marshal(simulationRunResults.Results)

			if !simulationRunResults.isValid() || !scalarm_worker.IsJSON(string(resultJson)) {
				fmt.Printf("[output.json] Invalid results.json: %s\n", resultJson)
				simulationRunResults.Status = "error"
				simulationRunResults.Results = nil
				simulationRunResults.Reason = fmt.Sprintf("Invalid results.json: %s", resultJson)
				resultJson = nil
			}

			// 4f. upload structural results of a simulation run
			data := url.Values{}
			data.Set("status", simulationRunResults.Status)
			data.Add("reason", simulationRunResults.Reason)
			data.Add("result", string(resultJson))

			fmt.Printf("[SiM] Results: %v\n", data)

			_, err = em.MarkSimulationRunAsComplete(simulation_index, data)
			if err != nil {
				fmt.Println("[SiM] Error during marking simulation run as complete.")
				Fatal(err)
			}

			// 4g. upload binary output if provided
			if _, err := os.Stat("output.tar.gz"); err == nil {
				fmt.Printf("[SiM] Uploading 'output.tar.gz' ...\n")
				file, err := os.Open("output.tar.gz")

				if err != nil {
					Fatal(err)
				}

				defer file.Close()

				requestBody := &bytes.Buffer{}
				writer := multipart.NewWriter(requestBody)
				part, err := writer.CreateFormFile("file", filepath.Base("output.tar.gz"))
				if err != nil {
					Fatal(err)
				}
				_, err = io.Copy(part, file)

				err = writer.Close()
				if err != nil {
					Fatal(err)
				}

				binariesUploadUrl := fmt.Sprintf("experiments/%s/simulations/%v", experimentId, simulation_index)
				binariesUploadUrlInfo := RequestInfo{"PUT", requestBody, writer.FormDataContentType(), binariesUploadUrl}
				body := ExecuteScalarmRequest(binariesUploadUrlInfo, storageManagers, config, client, communicationTimeout)

				fmt.Printf("[SiM] Response body: %s\n", body)
			}

			// 4h. upload stdout if provided
			if _, err := os.Stat("_stdout.txt"); err == nil {
				fmt.Println("[SiM] Uploading STDOUT of the simulation run ...")

				file, err := os.Open("_stdout.txt")
				if err != nil {
					Fatal(err)
				}

				requestBody := &bytes.Buffer{}
				writer := multipart.NewWriter(requestBody)
				part, err := writer.CreateFormFile("file", filepath.Base("_stdout.txt"))
				if err != nil {
					Fatal(err)
				}
				_, err = io.Copy(part, file)
				file.Close()

				err = writer.Close()
				if err != nil {
					Fatal(err)
				}

				stdoutUploadUrl := fmt.Sprintf("experiments/%s/simulations/%v/stdout", experimentId, simulation_index)
				stdoutUploadUrlInfo := RequestInfo{"PUT", requestBody, writer.FormDataContentType(), stdoutUploadUrl}
				body := ExecuteScalarmRequest(stdoutUploadUrlInfo, storageManagers, config, client, communicationTimeout)

				fmt.Printf("[SiM] Response body: %s\n", body)
			}

			// 5. clean up - removing simulation dir
			go func() {
				select {
				case _ = <-finished:
					os.RemoveAll(simulationDirPath)
					close(finished)
				}
			}()

			// 6. going to the root dir and moving
			if err = rootDir.Chdir(); err != nil {
				Fatal(err)
			}

			simulationsDone += 1

			if simulationsLimit > 0 {
				fmt.Printf("[SiM] Simulations done: %v/%v\n", simulationsDone, simulationsLimit)
			}

			if simulationsLimit > 0 && simulationsDone >= simulationsLimit {
				fmt.Printf("[SiM] Exiting due to simulation runs limit (%v)\n", simulationsLimit)
				os.Exit(1)
			}
		}
	}
}
