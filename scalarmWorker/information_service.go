package scalarmWorker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type InformationService struct {
	HttpClient           *http.Client
	BaseUrl              string
	CommunicationTimeout time.Duration
	Config               *SimulationManagerConfig
}

func (is *InformationService) GetExperimentManagers() ([]string, error) {
	iSReqInfo := RequestInfo{"GET", nil, "application/json", "experiment_managers"}

	resp, err := ExecuteScalarmRequest(iSReqInfo, []string{is.BaseUrl}, is.Config, is.HttpClient, is.CommunicationTimeout)

	if err != nil {
		return nil, err
	} else {
		return ParseInformationServiceResponse(resp)
	}
}

func (is *InformationService) GetStorageManagers() ([]string, error) {
	iSReqInfo := RequestInfo{"GET", nil, "application/json", "storage_managers"}

	resp, err := ExecuteScalarmRequest(iSReqInfo, []string{is.BaseUrl}, is.Config, is.HttpClient, is.CommunicationTimeout)

	if err != nil {
		return nil, err
	} else {
		return ParseInformationServiceResponse(resp)
	}
}

func ParseInformationServiceResponse(resp *http.Response) ([]string, error) {
	var experimentManagers []string

	if resp.StatusCode == 200 {
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		fmt.Printf("[SiM] Response body: %s.\n", body)

		if err := json.Unmarshal(body, &experimentManagers); err != nil {
			return nil, errors.New("Returned response body is not JSON.")
		}

		if len(experimentManagers) == 0 {
			return nil, errors.New("There is no Experiment Manager registered in Information Service. Please contact Scalarm administrators.")
		}

		return experimentManagers, nil

	} else if resp.StatusCode == 500 {

		return nil, errors.New("Information service response code: 500")

	} else {

		return nil, errors.New("Information service response code: " + strconv.Itoa(resp.StatusCode))

	}
}
