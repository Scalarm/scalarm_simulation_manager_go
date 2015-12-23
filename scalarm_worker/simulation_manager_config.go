package scalarm_worker

import "encoding/json"
import (
	"os"
	"errors"
)

// Config file description - this should be provided by Experiment Manager in 'config.json'
type SimulationManagerConfig struct {
	ExperimentId           string `json:"experiment_id"`
	InformationServiceUrl  string `json:"information_service_url"`
	ExperimentManagerUser  string `json:"experiment_manager_user"`
	ExperimentManagerPass  string `json:"experiment_manager_pass"`
	Development            bool   `json:"development"`
	StartAt                string `json:"start_at"`
	Timeout                int    `json:"timeout"`
	ScalarmCertificatePath string `json:"scalarm_certificate_path"`
	InsecureSSL            bool   `json:"insecure_ssl"`
}

func CreateSimulationManagerConfig(filePath string) (*SimulationManagerConfig, error) {
	configFile, err := os.Open(filePath)
	if err != nil {
		return nil, errors.New("Could not open file " + filePath +".")
	}

	config := new(SimulationManagerConfig)
	err = json.NewDecoder(configFile).Decode(&config)
	configFile.Close()

	if err != nil {
		return nil, errors.New("Incorrect JSON in the file.")
	}

	if config.Timeout <= 0 {
		config.Timeout = 60
	}

	return config, nil
}

