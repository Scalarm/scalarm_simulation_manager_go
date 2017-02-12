package scalarmWorker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type ExperimentManager struct {
	HttpClient           *http.Client
	BaseUrls             []string
	CommunicationTimeout time.Duration
	Config               *SimulationManagerConfig
	Username             string
	Password             string
	ExperimentId         string
}

func (em *ExperimentManager) GetNextSimulationRunConfig() (map[string]interface{}, error) {
	nextSimulationRunConfig := map[string]interface{}{}

	path := "experiments/" + em.ExperimentId + "/next_simulation"
	reqInfo := RequestInfo{"GET", nil, "", path}

	resp, err := ExecuteScalarmRequest(reqInfo, em.BaseUrls, em.Config, em.HttpClient, em.CommunicationTimeout)

	if err != nil {
		return nil, err
	} else {
		if resp.StatusCode == 200 {
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			if err := json.Unmarshal(body, &nextSimulationRunConfig); err != nil {
				return nil, errors.New("Returned response body is not JSON.")
			}

			return nextSimulationRunConfig, nil

		} else if resp.StatusCode == 500 {
			return nil, errors.New("Experiment manager response code: 500")
		} else {
			return nil, errors.New("Experiment manager response code: " + strconv.Itoa(resp.StatusCode))
		}
	}
}

func (em *ExperimentManager) MarkSimulationRunAsComplete(simulationIndex int, runResult url.Values) (map[string]interface{}, error) {
	emResponse := map[string]interface{}{}

	path := "experiments/" + em.ExperimentId + "/simulations/" + strconv.Itoa(simulationIndex) + "/mark_as_complete"
	reqInfo := RequestInfo{"POST", strings.NewReader(runResult.Encode()), "application/x-www-form-urlencoded", path}

	resp, err := ExecuteScalarmRequest(reqInfo, em.BaseUrls, em.Config, em.HttpClient, em.CommunicationTimeout)

	if err != nil {
		return nil, err
	} else {
		if resp.StatusCode == 200 {
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			if err := json.Unmarshal(body, &emResponse); err != nil {
				fmt.Printf("Receiving: %v\n", body)
				return nil, errors.New("Returned response body is not JSON.")
			}

			if statusVal, ok := emResponse["status"]; ok {
				if statusVal.(string) != "ok" && statusVal.(string) != "preconditioned_failed" {
					if reasonVal, ok := emResponse["reason"]; ok {
						return nil, errors.New(reasonVal.(string))
					}

					return nil, errors.New("Something went wrong but without any details")
				}
			}

			return emResponse, nil

		} else if resp.StatusCode == 500 {

			return nil, errors.New("Experiment manager response code: 500")

		} else {

			return nil, errors.New("Experiment manager response code: " + strconv.Itoa(resp.StatusCode))

		}
	}
}

func (em *ExperimentManager) DownloadExperimentCodeBase(codeBaseDir string) error {
	var responseBody []byte

	w, err := os.Create(path.Join(codeBaseDir, "code_base.zip"))
	if err != nil {
		return err
	}
	defer w.Close()

	codeBaseURL := "experiments/" + em.ExperimentId + "/code_base"
	codeBaseInfo := RequestInfo{"GET", nil, "", codeBaseURL}

	resp, err := ExecuteScalarmRequest(codeBaseInfo, em.BaseUrls, em.Config, em.HttpClient, em.CommunicationTimeout)
	if err != nil {
		return err
	}

	responseBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if _, err = io.Copy(w, bytes.NewReader(responseBody)); err != nil {
		return err
	}

	return nil
}

func (em *ExperimentManager) PostProgressInfo(simulationIndex int, results url.Values) error {
	emResponse := map[string]interface{}{}

	progressInfoPath := "experiments/" + em.ExperimentId + "/simulations/" + strconv.Itoa(simulationIndex) + "/progress_info"
	reqInfo := RequestInfo{"POST", strings.NewReader(results.Encode()), "application/x-www-form-urlencoded", progressInfoPath}

	resp, err := ExecuteScalarmRequest(reqInfo, em.BaseUrls, em.Config, em.HttpClient, em.CommunicationTimeout)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("Experiment manager response code: " + strconv.Itoa(resp.StatusCode))
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, &emResponse); err != nil {
		return errors.New("Returned response body is not JSON.")
	}

	if statusVal, ok := emResponse["status"]; ok {
		if statusVal.(string) != "ok" {
			if reasonVal, ok := emResponse["reason"]; ok {
				return errors.New(reasonVal.(string))
			}

			return errors.New("Something went wrong but without any details")
		}
	}

	return nil
}

func (em *ExperimentManager) ReportHostInfo(simulationIndex int, hostInfo *HostInfo) error {
	jsonStr, _ := json.Marshal(hostInfo)
	requestData := url.Values{}
	requestData.Set("host_info", string(jsonStr))

	url := "experiments/" + em.ExperimentId + "/simulations/" + strconv.Itoa(simulationIndex) + "/host_info"
	reqInfo := RequestInfo{"POST", strings.NewReader(requestData.Encode()), "application/x-www-form-urlencoded", url}

	resp, err := ExecuteScalarmRequest(reqInfo, em.BaseUrls, em.Config, em.HttpClient, em.CommunicationTimeout)
	defer resp.Body.Close()

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("Experiment manager response code: " + strconv.Itoa(resp.StatusCode))
	}

	return nil
}

func (em *ExperimentManager) ReportPerformanceStats(simulationIndex int, perfStats *PerformanceStats) error {
	jsonStr, _ := json.Marshal(perfStats)
	requestData := url.Values{}
	requestData.Set("stats", string(jsonStr))

	url := "experiments/" + em.ExperimentId + "/simulations/" + strconv.Itoa(simulationIndex) + "/performance_stats"
	reqInfo := RequestInfo{"POST", strings.NewReader(requestData.Encode()), "application/x-www-form-urlencoded", url}

	resp, err := ExecuteScalarmRequest(reqInfo, em.BaseUrls, em.Config, em.HttpClient, em.CommunicationTimeout)
	defer resp.Body.Close()

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("Experiment manager response code: " + strconv.Itoa(resp.StatusCode))
	}

	return nil
}
