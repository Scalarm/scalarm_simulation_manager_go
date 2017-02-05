package scalarmWorker

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"
	//	"io/ioutil"
	"errors"
)

type RequestInfo struct {
	HttpMethod    string
	Body          io.Reader
	ContentType   string
	ServiceMethod string
}

func Fatal(err error) {
	fmt.Printf("[Fatal error] %s\n", err.Error())
	os.Exit(1)
}

func ExecuteScalarmRequest(reqInfo RequestInfo, serviceUrls []string, config *SimulationManagerConfig,
	client *http.Client, timeout time.Duration) (*http.Response, error) {

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

		req.Header.Set("Accept", "application/json")

		if reqInfo.Body != nil {
			req.Header.Set("Content-Type", reqInfo.ContentType)
		}
		// 3. execute request with timeout
		response, err := GetWithTimeout(client, req, timeout)
		// 4. if response body is nil go to 2.
		if err == nil {
			return response, nil
		}
	}

	return nil, errors.New("Could not execute request against Scalarm service")
}

// Calling Get multiple time until valid response or exceed 'communicationTimeout' period
func GetWithTimeout(client *http.Client, request *http.Request, communicationTimeout time.Duration) (*http.Response, error) {
	var resp *http.Response
	var err error
	communicationFailed := true
	communicationStart := time.Now()

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

	return resp, nil
}
