package scalarmWorker

import "testing"

func TestHandlingCorrectJSON(t *testing.T) {
	correct_json := "{\"a\":1,\"b\":2}"

	if !IsJSON(correct_json) {
		t.Errorf("Correct JSON has been reported as incorrect")
	}
}

func TestHandlingInCorrectJSON(t *testing.T) {
	incorrect_json := "{\"a\":1,\"b\":2"

	if IsJSON(incorrect_json) {
		t.Errorf("Incorrect JSON has been reported as correct")
	}
}
