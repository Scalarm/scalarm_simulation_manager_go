package scalarmWorker

import (
	"errors"
	"testing"

	pscpu "github.com/shirou/gopsutil/cpu"
	pshost "github.com/shirou/gopsutil/host"
	psproc "github.com/shirou/gopsutil/process"
)

func fakeHostInfo() (*pshost.InfoStat, error) {
	return &pshost.InfoStat{
		OS:                   "linux",
		Platform:             "ubuntu",
		PlatformFamily:       "debian",
		PlatformVersion:      "16.10",
		KernelVersion:        "4.8.0-34-generic",
		VirtualizationSystem: "kvm",
		VirtualizationRole:   "host",
	}, nil
}

func fakeHostInfoWithError() (*pshost.InfoStat, error) {
	return nil, errors.New("randomHostError")
}

func fakeCPUInfo() ([]pscpu.InfoStat, error) {
	return []pscpu.InfoStat{
		pscpu.InfoStat{
			CPU:       1,
			VendorID:  "vendor",
			Family:    "family",
			Model:     "model",
			Stepping:  3,
			ModelName: "modelName",
			Mhz:       1.0,
			CacheSize: 64,
			Flags:     []string{"flag1", "flag2"},
		},
	}, nil
}

func fakeCPUInfoWithError() ([]pscpu.InfoStat, error) {
	return nil, errors.New("randomCPUError")
}

func fakeIoCounters(process *psproc.Process) (*psproc.IOCountersStat, error) {
	return &psproc.IOCountersStat{
		ReadCount:  1,
		WriteCount: 2,
		ReadBytes:  3,
		WriteBytes: 4,
	}, nil
}

func fakeIoCountersWithError(process *psproc.Process) (*psproc.IOCountersStat, error) {
	return nil, errors.New("randomIoError")
}

func fakeMemoryStats(process *psproc.Process) (*psproc.MemoryInfoStat, error) {
	return &psproc.MemoryInfoStat{
		RSS:  1,
		VMS:  2,
		Swap: 3,
	}, nil
}

func fakeMemoryStatsWithError(process *psproc.Process) (*psproc.MemoryInfoStat, error) {
	return nil, errors.New("randompMemoryError")
}

func fakeTimesStats(process *psproc.Process) (*pscpu.TimesStat, error) {
	return &pscpu.TimesStat{
		User:   1.0,
		System: 2.0,
		Iowait: 3.0,
	}, nil
}

func fakeTimesStatsWithError(process *psproc.Process) (*pscpu.TimesStat, error) {
	return nil, errors.New("randomCPUError")
}

// ACTUAL TESTS

func TestExtractingHostInfoShouldReturnFilledStructWhenNoErrorsOccur(t *testing.T) {
	ps := new(PsUtil)
	ps.getHostInfo = fakeHostInfo
	ps.getCPUInfo = fakeCPUInfo

	hostInfo, err := ExtractHostInfo(ps)

	if err != nil {
		t.Errorf("Got: '%v' - Expected nil", err)
	}

	if hostInfo.Cores != 1 {
		t.Errorf("Got: '%v' - Expected '1'", hostInfo.Cores)
	}

	if hostInfo.OS != "linux" {
		t.Errorf("Got: '%v' - Expected 'linux'", hostInfo.OS)
	}

	if hostInfo.KernelVersion != "4.8.0-34-generic" {
		t.Errorf("Got: '%v' - Expected '4.8.0-34-generic'", hostInfo.KernelVersion)
	}

	if hostInfo.VendorID != "vendor" {
		t.Errorf("Got: '%v' - Expected 'vendor'", hostInfo.VendorID)
	}
}

func TestExtractHostShouldReturnHostErrorWhenHostErrorOccurs(t *testing.T) {
	ps := new(PsUtil)
	ps.getHostInfo = fakeHostInfoWithError
	ps.getCPUInfo = fakeCPUInfo

	_, err := ExtractHostInfo(ps)

	if err == nil {
		t.Errorf("Got: 'nil' - Expected 'randomHostError'")
	}

	if err.Error() != "randomHostError" {
		t.Errorf("Got: '%v' - Expected 'randomHostError'", err.Error())
	}
}

func TestExtractHostShouldReturnCPUErrorWhenCPUErrorOccurs(t *testing.T) {
	ps := new(PsUtil)
	ps.getHostInfo = fakeHostInfo
	ps.getCPUInfo = fakeCPUInfoWithError

	_, err := ExtractHostInfo(ps)

	if err == nil {
		t.Errorf("Got: 'nil' - Expected 'randomCPUError'")
	}

	if err.Error() != "randomCPUError" {
		t.Errorf("Got: '%v' - Expected 'randomCPUError'", err.Error())
	}
}

func TestExtractPerformanceStatsShouldReturnFilledStructWhenNoErrorsOccur(t *testing.T) {
	ps := new(PsUtil)
	ps.getCPUTimes = fakeTimesStats
	ps.getMemoryStats = fakeMemoryStats
	ps.getIoStats = fakeIoCounters

	proc, _ := psproc.NewProcess(1)
	perfStats, err := ExtractPerformanceStats(proc, ps)

	if err != nil {
		t.Errorf("Got: '%v' - Expected nil", err)
	}

	if perfStats.Utime != 1.0 {
		t.Errorf("Got: '%v' - Expected '1.0'", perfStats.Utime)
	}

	if perfStats.Vms != 2 {
		t.Errorf("Got: '%v' - Expected '2'", perfStats.Vms)
	}

	if perfStats.WriteBytes != 4 {
		t.Errorf("Got: '%v' - Expected '4'", perfStats.WriteBytes)
	}
}

func TestExtractPerformanceStatsShouldReturnTimesErrorWhenTimeErrorsOccur(t *testing.T) {
	ps := new(PsUtil)
	ps.getCPUTimes = fakeTimesStatsWithError
	ps.getMemoryStats = fakeMemoryStats
	ps.getIoStats = fakeIoCounters

	proc, _ := psproc.NewProcess(1)
	_, err := ExtractPerformanceStats(proc, ps)

	if err == nil {
		t.Errorf("Got: 'nil' - Expected 'randomCPUError'")
	}

	if err.Error() != "randomCPUError" {
		t.Errorf("Got: '%v' - Expected 'randomCPUError'", err.Error())
	}
}

func TestExtractPerformanceStatsShouldReturnMemErrorWhenMemErrorsOccur(t *testing.T) {
	ps := new(PsUtil)
	ps.getCPUTimes = fakeTimesStats
	ps.getMemoryStats = fakeMemoryStatsWithError
	ps.getIoStats = fakeIoCounters

	proc, _ := psproc.NewProcess(1)
	_, err := ExtractPerformanceStats(proc, ps)

	if err == nil {
		t.Errorf("Got: 'nil' - Expected 'randompMemoryError'")
	}

	if err.Error() != "randompMemoryError" {
		t.Errorf("Got: '%v' - Expected 'randompMemoryError'", err.Error())
	}
}

func TestExtractPerformanceStatsShouldReturnIoErrorWhenIoErrorsOccur(t *testing.T) {
	ps := new(PsUtil)
	ps.getCPUTimes = fakeTimesStats
	ps.getMemoryStats = fakeMemoryStats
	ps.getIoStats = fakeIoCountersWithError

	proc, _ := psproc.NewProcess(1)
	_, err := ExtractPerformanceStats(proc, ps)

	if err == nil {
		t.Errorf("Got: 'nil' - Expected 'randomIoError'")
	}

	if err.Error() != "randomIoError" {
		t.Errorf("Got: '%v' - Expected 'randomIoError'", err.Error())
	}
}

func TestCollectPerformanceStatsShouldReturnErrorWhenNoProcessWaFound(t *testing.T) {
	ps := new(PsUtil)
	ps.getCPUTimes = fakeTimesStats
	ps.getMemoryStats = fakeMemoryStats
	ps.getIoStats = fakeIoCounters

	_, err := CollectPerformanceStats(32000, ps)

	if err == nil {
		t.Errorf("Got: 'nil' - Expected 'could not create a proc with id error'")
	}

	if err.Error() != "Could not create process with pid 32000" {
		t.Errorf("Got: '%v' - Expected 'Could not create process with pid 32000'", err.Error())
	}
}
