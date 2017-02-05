package scalarmWorker

import (
	"testing"
)

func TestHandlingCorrectSimulationManagerConfig(t *testing.T) {
	config, err := CreateSimulationManagerConfig("test_assets/correct_input.json")

	if err != nil {
		t.Errorf("Got: '%v' - Expected nil", err)
	}

	expected_id := "54e4d4fd4269a870f7004b01"
	if config.ExperimentId != expected_id {
		t.Errorf("Got: '%v' - Expected '%v'", config.ExperimentId, expected_id)
	}
}

func TestHandlingCorrectSimulationManagerConfigWithoutTimeout(t *testing.T) {
	config, err := CreateSimulationManagerConfig("test_assets/correct_input.json")

	if err != nil {
		t.Errorf("Got: '%v' - Expected nil", err)
	}

	expected_timeout := 60
	if config.Timeout != expected_timeout {
		t.Errorf("Got: '%v' - Expected '%v'", config.Timeout, expected_timeout)
	}
}

func TestHandlingNoFileToCreateSimulationManagerConfig(t *testing.T) {
	_, err := CreateSimulationManagerConfig("test_assets/does_not_exist.json")
	if err == nil {
		t.Errorf("Got: nil - Expected not nil", err)
	}

	expected_msg := "Could not open file test_assets/does_not_exist.json."

	if err.Error() != expected_msg {
		t.Errorf("Got: '%v' - Expected '%v'", err.Error(), expected_msg)
	}
}

func TestHandlingIncorrectSimulationManagerConfig(t *testing.T) {
	_, err := CreateSimulationManagerConfig("test_assets/incorrect_input.json")
	if err == nil {
		t.Errorf("Got: nil - Expected not nil", err)
	}

	expected_msg := "Incorrect JSON in the file."

	if err.Error() != expected_msg {
		t.Errorf("Got: '%v' - Expected '%v'", err.Error(), expected_msg)
	}
}
