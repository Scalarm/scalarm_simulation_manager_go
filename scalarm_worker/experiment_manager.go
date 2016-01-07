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
)

type ExperimentManager struct {
	HttpClient           *http.Client
	BaseUrl              string
	CommunicationTimeout time.Duration
	Config               *SimulationManagerConfig
	Username             string
	Password             string
}

func (em *ExperimentManager) GetNextSimulationRunConfig(experiment_id string) (map[string]interface{}, error) {
	nextSimulationRunConfig := map[string]interface{}{}

	path := "experiments/" + experiment_id + "/next_simulation"
	reqInfo := RequestInfo{"GET", nil, "", path}

	resp, err := ExecuteScalarmRequest(reqInfo, []string{em.BaseUrl}, em.Config, em.HttpClient, em.CommunicationTimeout)

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

func (em *ExperimentManager) MarkSimulationRunAsComplete(experiment_id string, simulation_index int, runResult url.Values) (map[string]interface{}, error) {
  emResponse := map[string]interface{} {}

  path := "experiments/" + experiment_id + "/simulations/" + strconv.Itoa(simulation_index) + "/mark_as_complete"
	reqInfo := RequestInfo{"POST", nil, "application/x-www-form-urlencoded", path}

	resp, err := ExecuteScalarmRequest(reqInfo, []string{em.BaseUrl}, em.Config, em.HttpClient, em.CommunicationTimeout)

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
