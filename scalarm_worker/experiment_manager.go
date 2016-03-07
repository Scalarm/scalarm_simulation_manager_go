package scalarm_worker

import (
	"io/ioutil"
	"net/http"
	"time"
	// "fmt"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
  "strings"
	"os"
	"path"
	"io"
	"bytes"
)

type ExperimentManager struct {
	HttpClient           *http.Client
	BaseUrls             []string
	CommunicationTimeout time.Duration
	Config               *SimulationManagerConfig
	Username             string
	Password             string
	ExperimentId				 string
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
  emResponse := map[string]interface{} {}

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
        return nil, errors.New("Returned response body is not JSON.")
      }

      if statusVal, ok := emResponse["status"]; ok {
        if statusVal.(string) != "ok" {
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

func (em *ExperimentManager) DownloadExperimentCodeBase(codeBaseDir string) (error) {
	var responseBody []byte

	w, err := os.Create(path.Join(codeBaseDir, "code_base.zip"))
	if err != nil {
		return err
	}
	defer w.Close()

	codeBaseUrl := "experiments/" + em.ExperimentId + "/code_base"
	codeBaseInfo := RequestInfo{"GET", nil, "", codeBaseUrl}

	resp, err := ExecuteScalarmRequest(codeBaseInfo, em.BaseUrls, em.Config, em.HttpClient, em.CommunicationTimeout)

	responseBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if _, err = io.Copy(w, bytes.NewReader(responseBody)); err != nil {
		return err
	}

	return nil
}

func (em *ExperimentManager) PostProgressInfo(simulationIndex int, results url.Values) (error) {
	emResponse := map[string]interface{} {}

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