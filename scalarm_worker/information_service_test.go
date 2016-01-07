package scalarm_worker

import (
	"testing"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"time"
)

// =========== UTILS/SETUP ===========

func setupInformationService(config *SimulationManagerConfig, client *http.Client) InformationService {
	return InformationService{
			HttpClient: client,
			BaseUrl: "system.scalarm.com/information",
			CommunicationTimeout: 10 * time.Second,
			Config: config}
}

// =========== =========== ===========

func TestInformationServiceShouldReturnListOfEmAddressesWhenEverythingIsOk(t *testing.T) {
	// === GIVEN ===
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `["siteA.com", "siteB.com"]`)
	}))
	defer server.Close()

	is := setupInformationService(getSimConfig(), getHttpClientMock(server.URL))

	// === WHEN ===
	experimentManagers, err := is.GetExperimentManagers()

	// === THEN ===
	if err != nil {
		t.Errorf("Returned error should be nil, but it is '%v'", err)
	}

	expected := []string{"siteA.com", "siteB.com"}

	if !reflect.DeepEqual(expected, experimentManagers) {
		t.Errorf("Returned list of experiment managers is not what we expected to be. Actual: %v, Expected: %v",
			experimentManagers, expected)
	}
}

func TestInformationServiceShouldReturnErrorWhen500(t *testing.T) {
	// === GIVEN ===
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprintln(w, `some loooong html text`)
	}))
	defer server.Close()

	is := setupInformationService(getSimConfig(), getHttpClientMock(server.URL))

	// === WHEN ===
	experimentManagers, err := is.GetExperimentManagers()

	// === THEN ===
	if experimentManagers != nil {
		t.Errorf("Got: '%v' - Expected nil", experimentManagers)
	}

	if err == nil {
		t.Errorf("Error expected but got nil")
	}

	expected_error := "Information service response code: 500"
	if err.Error() != expected_error {
		t.Errorf("Got: '%v' - Expected '%v'", err.Error(), expected_error)
	}
}

func TestInformationServiceShouldErrorWhenNoServiceRegistered(t *testing.T) {
	// === GIVEN ===
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `[]`)
	}))
	defer server.Close()

	is := setupInformationService(getSimConfig(), getHttpClientMock(server.URL))

	// === WHEN ===
	experimentManagers, err := is.GetExperimentManagers()

	// === THEN ===
	if experimentManagers != nil {
		t.Errorf("Got: '%v' - Expected nil", experimentManagers)
	}

	if err == nil {
		t.Errorf("Error expected but got nil")
	}

	expected_error := "There is no Experiment Manager registered in Information Service. Please contact Scalarm administrators."
	if err.Error() != expected_error {
		t.Errorf("Got: '%v' - Expected '%v'", err.Error(), expected_error)
	}
}

func TestInformationServiceShouldErrorWhenIncorrectJSONInReturn(t *testing.T) {
	// === GIVEN ===
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `"siteA.com", "siteB.com"`)
	}))
	defer server.Close()

	is := setupInformationService(getSimConfig(), getHttpClientMock(server.URL))

	// === WHEN ===
	experimentManagers, err := is.GetExperimentManagers()

	// === THEN ===
	if experimentManagers != nil {
		t.Errorf("Got: '%v' - Expected nil", experimentManagers)
	}

	if err == nil {
		t.Errorf("Error expected but got nil")
	}

	expected_error := "Returned response body is not JSON."
	if err.Error() != expected_error {
		t.Errorf("Got: '%v' - Expected '%v'", err.Error(), expected_error)
	}
}

func TestInformationServiceShouldErrorNoServiceAvailable(t *testing.T) {
	// === GIVEN ===
	config := &SimulationManagerConfig{
				ExperimentManagerUser: "user",
				ExperimentManagerPass: "pass",
				Development: true,
			}

	client := &http.Client{}

	is := InformationService{
				HttpClient: client,
				BaseUrl: "someveryincorrecturl",
				CommunicationTimeout: 5 * time.Second,
				Config: config}

	// === WHEN ===
	experimentManagers, err := is.GetExperimentManagers()

	// === THEN ===
	if experimentManagers != nil {
		t.Errorf("Got: '%v' - Expected nil", experimentManagers)
	}

	if err == nil {
		t.Errorf("Error expected but got nil")
	}

	expected_error := "Could not execute request against Scalarm service"
	if err.Error() != expected_error {
		t.Errorf("Got: '%v' - Expected '%v'", err.Error(), expected_error)
	}
}
