package scalarm_worker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	// "reflect"
	"time"
)

// =========== UTILS/SETUP ===========

func setupExperimentManager(config *SimulationManagerConfig, client *http.Client) ExperimentManager {
	return ExperimentManager{
		HttpClient:           client,
		BaseUrls:              []string{"system.scalarm.com"},
		CommunicationTimeout: 5 * time.Second,
		Config:               config,
		ExperimentId:					"568e5bece138232e76000002"}
}

// =========== =========== ===========

func TestExperimentManagerShouldAskForJsonFormatWhenRequestingNextSimulation(t *testing.T) {
	// === GIVEN ===
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" ||
			r.Header.Get("Accept") != "application/json" ||
			r.URL.Path != "/experiments/568e5bece138232e76000002/next_simulation" {
			w.WriteHeader(500)
			fmt.Fprintln(w, `<div>blebleble</div>`)

			return
		}

		w.WriteHeader(200)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok","simulation_id":1,"execution_constraints":{"time_constraint_in_sec":3300},"input_parameters":{"parameter1":0.0,"parameter2":-100.0}}`)
	}))
	defer server.Close()

	em := setupExperimentManager(getSimConfig(), getHttpClientMock(server.URL))

	// === WHEN ===
	nextSimulationRunConfig, err := em.GetNextSimulationRunConfig()

	// === THEN ===
	if err != nil {
		t.Errorf("Returned error should be nil, but it is '%v'", err)
    return
	}

  if nextSimulationRunConfig["status"].(string) != "ok" {
		t.Errorf("Returned next simulation run config is what we expected to be. Actual: %v, Expected: %v",
		    nextSimulationRunConfig, "ok")
	}
}

func TestExperimentManagerShouldAskForJsonFormatWhenSendingMarkAsComplete(t *testing.T) {
	// === GIVEN ===
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" ||
			r.Header.Get("Accept") != "application/json" ||
			r.URL.Path != "/experiments/568e5bece138232e76000002/simulations/1/mark_as_complete" {
			w.WriteHeader(500)
			fmt.Fprintln(w, `<div>blebleble</div>`)

			return
		}

		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	em := setupExperimentManager(getSimConfig(), getHttpClientMock(server.URL))
	simRunResultJson, _ := json.Marshal(map[string]int{
    "x": 1,
  })

	simRunResult := url.Values{}
	simRunResult.Set("status", "ok")
	simRunResult.Add("reason", "")
	simRunResult.Add("result", string(simRunResultJson))

	fmt.Printf("[SiM] Results: %v\n", simRunResult)

	// === WHEN ===
	resp, err := em.MarkSimulationRunAsComplete(1, simRunResult)

	// === THEN ===
	if err != nil {
		t.Errorf("Returned error should be nil, but it is '%v'", err)
    return
	}

  if resp["status"].(string) != "ok" {
		t.Errorf("Returned next simulation run config is what we expected to be. Actual: %v, Expected: %v",
			resp, "ok")
	}

}

func TestExperimentManagerShouldHandleHttpErrorsWhenSendingMarkAsComplete(t *testing.T) {
	// === GIVEN ===
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprintln(w, `<div>blebleble</div>`)
	}))
	defer server.Close()

	em := setupExperimentManager(getSimConfig(), getHttpClientMock(server.URL))
	simRunResultJson, _ := json.Marshal(map[string]int{
    "x": 1,
  })

	simRunResult := url.Values{}
	simRunResult.Set("status", "ok")
	simRunResult.Add("reason", "")
	simRunResult.Add("result", string(simRunResultJson))

	fmt.Printf("[SiM] Results: %v\n", simRunResult)

	// === WHEN ===
	_, err := em.MarkSimulationRunAsComplete(1, simRunResult)

	// === THEN ===
	if err == nil {
		t.Errorf("Returned error should be 'Experiment manager response code: 500', but it is nil")
    return
	}
}

func TestExperimentManagerShouldHandleScalarmErrorsWhenSendingMarkAsComplete(t *testing.T) {
	// === GIVEN ===
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, `{"status":"error", "reason":"Something went wrong"}`)
	}))
	defer server.Close()

	em := setupExperimentManager(getSimConfig(), getHttpClientMock(server.URL))
	simRunResultJson, _ := json.Marshal(map[string]int{
    "x": 1,
  })

	simRunResult := url.Values{}
	simRunResult.Set("status", "ok")
	simRunResult.Add("reason", "")
	simRunResult.Add("result", string(simRunResultJson))

	// === WHEN ===
	_, err := em.MarkSimulationRunAsComplete(1, simRunResult)

  expectedError := "Something went wrong"

	// === THEN ===
	if err == nil {
		t.Errorf("Returned error should be '%v', but it is %v", expectedError, err)
    return
	}

  if err.Error() != expectedError {
    t.Errorf("Returned error should be '%v', but it is %v", expectedError, err)
    return
  }
}

func TestExperimentManagerShouldHandleScalarmErrorsWithoutReasonsWhenSendingMarkAsComplete(t *testing.T) {
	// === GIVEN ===
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, `{"status":"error"}`)
	}))
	defer server.Close()

	em := setupExperimentManager(getSimConfig(), getHttpClientMock(server.URL))
	simRunResultJson, _ := json.Marshal(map[string]int{
    "x": 1,
  })

	simRunResult := url.Values{}
	simRunResult.Set("status", "ok")
	simRunResult.Add("reason", "")
	simRunResult.Add("result", string(simRunResultJson))

	// === WHEN ===
	_, err := em.MarkSimulationRunAsComplete(1, simRunResult)

  expectedError := "Something went wrong but without any details"

	// === THEN ===
	if err == nil {
		t.Errorf("Returned error should be '%v', but it is %v", expectedError, err)
    return
	}

  if err.Error() != expectedError {
    t.Errorf("Returned error should be '%v', but it is %v", expectedError, err)
    return
  }
}
