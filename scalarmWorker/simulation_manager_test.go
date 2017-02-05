package scalarmWorker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

func TestSimRunShouldRunSimulationsFromExperiment(t *testing.T) {
	// === GIVEN ===
	os.RemoveAll("./experiment_1")
	// defer os.RemoveAll("./experiment_1")

	allSimulationsSent := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Responding to: %v\n", r.URL.Path)

		if r.URL.Path == "/information/experiment_managers" || r.URL.Path == "/information/storage_managers" {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `["siteA.com", "siteB.com"]`)
		} else if r.URL.Path == "/experiments/1/next_simulation" {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			if allSimulationsSent {
				fmt.Fprintln(w, `{"status":"all_sent","reason":"There is no more simulations"}`)
			} else {
				allSimulationsSent = true
				fmt.Fprintln(w, `{"status":"ok","simulation_id":1,"execution_constraints":{"time_constraint_in_sec":3300},"input_parameters":{"parameter1":10.0,"parameter2":2.0}}`)
			}
		} else if r.URL.Path == "/experiments/1/code_base" {
			http.ServeFile(w, r, "./test_assets/code_base.zip")
		} else if r.URL.Path == "/experiments/1/simulations/1/mark_as_complete" {
			err := r.ParseForm()
			if err != nil {
				panic(err)
			}

			params := r.PostFormValue("result")
			type Result struct {
				Product int `json:"product"`
			}
			res := new(Result)
			err = json.Unmarshal([]byte(params), res)

			if err != nil {
				w.WriteHeader(500)
			} else {
				fmt.Printf("Received: %v\n", params)
				if res.Product != 20 {
					fmt.Println("Something went wrong")
					w.WriteHeader(500)
				} else {
					fmt.Println("Everything went great")
					w.WriteHeader(200)
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprintln(w, `{"status":"ok"}`)
				}
			}

		} else if r.URL.Path == "/experiments/1/simulations/1/stdout" {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(500)
		}

	}))
	defer server.Close()

	// Make a transport that reroutes all traffic to the example server
	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			fmt.Printf("Calling: %v\n", req.URL.Path)
			return url.Parse(server.URL + "" + req.URL.Path)
		},
	}

	config := SimulationManagerConfig{
		ExperimentId:           "1",
		InformationServiceUrl:  "www.example.com/information",
		ExperimentManagerUser:  "user",
		ExperimentManagerPass:  "pass",
		Development:            true,
		Timeout:                30,
		ScalarmCertificatePath: "",
		InsecureSSL:            true,
	}

	wd, _ := os.Getwd()

	sim := SimulationManager{
		Config:      config,
		HttpClient:  &http.Client{Transport: transport},
		RootDirPath: wd,
	}

	sim.Run()
}
