package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"math/rand"
	"time"

	scalarmWorker "github.com/scalarm/scalarm_simulation_manager_go/scalarmWorker"
)

// VERSION current version of the app
const VERSION string = "17.02"

// Fatal utility function to log a fatal error
func Fatal(err error) {
	fmt.Printf("[Fatal error] %v\n", err)
	os.Exit(1)
}

func main() {
	fmt.Printf("[SiM] Scalarm Simulation Manager, version: %s\n", VERSION)
	rand.Seed(time.Now().UTC().UnixNano())

	// 0. remember current location
	rootDirPath, _ := os.Getwd()
	fmt.Printf("[SiM] working directory: %s\n", rootDirPath)

	// 1. load config file
	config, err := scalarmWorker.CreateSimulationManagerConfig("config.json")
	if err != nil {
		Fatal(err)
	}

	// 2. prepare HTTP client
	var client *http.Client
	tlsConfig := tls.Config{InsecureSkipVerify: config.InsecureSSL}

	if config.ScalarmCertificatePath != "" {
		CAPool := x509.NewCertPool()
		severCert, err := ioutil.ReadFile(config.ScalarmCertificatePath)
		if err != nil {
			Fatal(fmt.Errorf("Could not load Scalarm certificate"))
		}
		CAPool.AppendCertsFromPEM(severCert)

		tlsConfig.RootCAs = CAPool
	}

	client = &http.Client{Transport: &http.Transport{TLSClientConfig: &tlsConfig}}

	// 3. create simulation manager instance and run it

	sim := scalarmWorker.SimulationManager{
		Config:      config,
		HttpClient:  client,
		RootDirPath: rootDirPath,
	}

	sim.Run()
}
