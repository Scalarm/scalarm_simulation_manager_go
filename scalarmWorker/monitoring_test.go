package scalarmWorker

import (
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	pscpu "github.com/shirou/gopsutil/cpu"
	pshost "github.com/shirou/gopsutil/host"
	psproc "github.com/shirou/gopsutil/process"
)

func realPs() PsUtil {
	return PsUtil{
		getCPUTimes:    GetCPUTimes,
		getIoStats:     GetIOStats,
		getMemoryStats: GetMemoryStats,
	}
}

func startPerfTestScript(t *testing.T) *os.Process {
	testScriptCmd := exec.Command("sh", "test_assets/perfStatsProc.sh", "0")
	testScriptCmd.Start()
	t.Logf("Perf script started - PID: %d", testScriptCmd.Process.Pid)

	return testScriptCmd.Process
}

func killProcWithMaxPid(t *testing.T, procs map[int32]*PerformanceStats) {
	var firstPid, maxPid int32

	maxPid = 0

	for firstPid = range procs {
		if maxPid < firstPid {
			maxPid = firstPid
		}
	}

	t.Logf("Killing process with pid %d", maxPid)

	proc := os.Process{Pid: int(maxPid)}
	proc.Kill()
}

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

func killProcs(processes map[int32]*PerformanceStats) {
	for pid := range processes {
		proc := os.Process{Pid: int(pid)}
		proc.Kill()
	}
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

	if perfStats.Timestamp <= 0 {
		t.Errorf("Got: '%v' - Expected something more than 0", perfStats.Timestamp)
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

func TestCollectPerformanceStatsShouldReturnErrorWhenNoProcessWasFound(t *testing.T) {
	ps := new(PsUtil)
	ps.getCPUTimes = fakeTimesStats
	ps.getMemoryStats = fakeMemoryStats
	ps.getIoStats = fakeIoCounters

	_, err := CollectPerformanceStats(32000, ps)

	if err == nil {
		t.Errorf("Got: 'nil' - Expected 'could not create a proc with id error'")
	}

	expectedErrMsg := "Could not create process with pid 32000: open /proc/32000: no such file or directory"
	if err.Error() != expectedErrMsg {
		t.Errorf("Got: '%v' - Expected '%s'", err.Error(), expectedErrMsg)
	}
}

func TestCollectPerformanceStatsShouldReturnMapOfPidsAndPerformanceStats(t *testing.T) {
	ps := realPs()
	scriptProc := startPerfTestScript(t)

	stats, err := CollectPerformanceStats(scriptProc.Pid, &ps)
	defer killProcs(stats)

	if err != nil {
		t.Errorf("Got: '%v' - Expected nil", err)
	}

	if len(stats) != 3 {
		t.Errorf("Got: %d - Expected 3", len(stats))
	}
}

func TestAggregatePerformanceStatsMapsShouldAddInformationFromTwoMapsWithStats(t *testing.T) {
	ps := realPs()
	scriptProc := startPerfTestScript(t)

	stats1, _ := CollectPerformanceStats(scriptProc.Pid, &ps)
	defer killProcs(stats1)

	killProcWithMaxPid(t, stats1)

	stats2, _ := CollectPerformanceStats(scriptProc.Pid, &ps)

	aggregatedStats := AggregatePerformanceStatsMaps(stats1, stats2)

	if len(aggregatedStats) != 3 {
		t.Errorf("Got: %d - Expected 3", len(aggregatedStats))
	}
}

func TestAggregatePerformanceStatsShouldSumStatsFromMultipleProcesses(t *testing.T) {
	ps := realPs()
	scriptProc := startPerfTestScript(t)

	stats1, _ := CollectPerformanceStats(scriptProc.Pid, &ps)
	defer killProcs(stats1)

	aggregatedProcStats := AggregatePerformanceStats(stats1)

	utimeSum := float64(0)
	for _, procStats := range stats1 {
		utimeSum += procStats.Utime
	}

	if aggregatedProcStats.Utime != utimeSum {
		t.Errorf("Got: %f - Expected %f", aggregatedProcStats.Utime, utimeSum)
	}

	killProcWithMaxPid(t, stats1)

	time.Sleep(1 * time.Second)

	stats2, _ := CollectPerformanceStats(scriptProc.Pid, &ps)

	stats3 := AggregatePerformanceStatsMaps(stats1, stats2)
	aggregatedProcStats = AggregatePerformanceStats(stats3)

	utimeSum = float64(0)
	for _, procStats := range stats3 {
		utimeSum += procStats.Utime
	}

	if aggregatedProcStats.Utime != utimeSum {
		t.Errorf("Got: %f - Expected %f", aggregatedProcStats.Utime, utimeSum)
	}

	if aggregatedProcStats.ProcessCount != 3 {
		t.Errorf("Got: %d - Expected 3", aggregatedProcStats.ProcessCount)
	}

	if aggregatedProcStats.Timestamp <= 0 {
		t.Errorf("Got: '%v' - Expected something more than 0", aggregatedProcStats.Timestamp)
	}

}
