package scalarmWorker

// Results structure - we send this back to Experiment Manager
type SimulationRunResults struct {
	Status  string      `json:"status"`
	Results interface{} `json:"results"`
	Reason  string      `json:"reason"`
}

func (res *SimulationRunResults) isValid() bool {
	return (res.Status == "ok" && res.Results != nil) || (res.Status == "error" && res.Reason != "")
}
