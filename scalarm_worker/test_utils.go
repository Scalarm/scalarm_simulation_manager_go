package scalarm_worker

import (
	"net/http"
	"net/url"
)

// =========== UTILS/SETUP ===========

func getSimConfig() (*SimulationManagerConfig) {
	return &SimulationManagerConfig{
			ExperimentManagerUser: "user",
			ExperimentManagerPass: "pass",
			Development: true,
		}
}

func getHttpClientMock(testServerUrl string) (*http.Client) {

	// Make a transport that reroutes all traffic to the example server
	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(testServerUrl)
		},
	}

	return &http.Client{Transport: transport}
}
