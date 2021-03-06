package scalarmWorker

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	pscpu "github.com/shirou/gopsutil/cpu"
	pshost "github.com/shirou/gopsutil/host"
	psproc "github.com/shirou/gopsutil/process"
)

var newProcessFunc = psproc.NewProcess

// PsUtil is a struct, which includes two ps functions: host and cpu related
type PsUtil struct {
	getHostInfo    func() (*pshost.InfoStat, error)
	getCPUInfo     func() ([]pscpu.InfoStat, error)
	getIoStats     func(process *psproc.Process) (*psproc.IOCountersStat, error)
	getMemoryStats func(process *psproc.Process) (*psproc.MemoryInfoStat, error)
	getCPUTimes    func(process *psproc.Process) (*pscpu.TimesStat, error)
}

// GetIOStats gets IO-related stats using psutils
func GetIOStats(process *psproc.Process) (*psproc.IOCountersStat, error) {
	return process.IOCounters()
}

// GetMemoryStats gets memory-related stats using psutils
func GetMemoryStats(process *psproc.Process) (*psproc.MemoryInfoStat, error) {
	return process.MemoryInfo()
}

// GetCPUTimes gets CPU-related stats using psutils
func GetCPUTimes(process *psproc.Process) (*pscpu.TimesStat, error) {
	return process.Times()
}

// HostInfo contains essential information about the host on which SiM is running
type HostInfo struct {
	OS                   string   `json:"os"`              // ex: freebsd, linux
	Platform             string   `json:"platform"`        // ex: ubuntu, linuxmint
	PlatformFamily       string   `json:"platformFamily"`  // ex: debian, rhel
	PlatformVersion      string   `json:"platformVersion"` // version of the complete OS
	KernelVersion        string   `json:"kernelVersion"`   // version of the OS kernel (if available)
	VirtualizationSystem string   `json:"virtualizationSystem"`
	VirtualizationRole   string   `json:"virtualizationRole"` // guest or host
	Cores                int      `json:"cores"`
	VendorID             string   `json:"vendorId"`
	Family               string   `json:"family"`
	Model                string   `json:"model"`
	Stepping             int32    `json:"stepping"`
	ModelName            string   `json:"modelName"`
	Mhz                  float64  `json:"mhz"`
	CacheSize            int32    `json:"cacheSize"`
	Flags                []string `json:"flags"`
	Timestamp            int64    `json:"timestamp"`
}

// ExtractHostInfo Extract information about host on which the sim is running
func ExtractHostInfo(ps *PsUtil) (*HostInfo, error) {
	info := new(HostInfo)

	coreStats, err := ps.getCPUInfo()
	if err != nil {
		fmt.Printf("[pscpu.Info()] error: %v,\n", err)
		return nil, err
	}

	info.Timestamp = time.Now().Unix()

	info.Cores = len(coreStats)
	info.VendorID = coreStats[0].VendorID
	info.Family = coreStats[0].Family
	info.Model = coreStats[0].Model
	info.Stepping = coreStats[0].Stepping
	info.ModelName = coreStats[0].ModelName
	info.Mhz = coreStats[0].Mhz
	info.CacheSize = coreStats[0].CacheSize
	info.Flags = coreStats[0].Flags

	host, err := ps.getHostInfo()
	if err != nil {
		fmt.Printf("[pshost.Info()] error: %v,\n", err)
		return nil, err
	}

	info.OS = host.OS
	info.Platform = host.Platform
	info.PlatformFamily = host.PlatformFamily
	info.PlatformVersion = host.PlatformVersion
	info.KernelVersion = host.KernelVersion
	info.VirtualizationSystem = host.VirtualizationSystem
	info.VirtualizationRole = host.VirtualizationRole

	return info, nil
}

// PerformanceStats keeps basic performance-related information
type PerformanceStats struct {
	Timestamp int64 `json:"timestamp"`

	Utime  float64 `json:"utime"`
	Stime  float64 `json:"stime"`
	Iowait float64 `json:"iowait"`

	// in bytes
	Rss  uint64 `json:"rss"`
	Vms  uint64 `json:"vms"`
	Swap uint64 `json:"swap"`

	ReadCount  uint64 `json:"read_count"`
	WriteCount uint64 `json:"write_count"`

	// in bytes
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`

	ProcessCount uint64 `json:"process_count"`
}

func extractTimes(stats *PerformanceStats, cpuInfo *pscpu.TimesStat) {
	stats.Utime = cpuInfo.User
	stats.Stime = cpuInfo.System
	stats.Iowait = cpuInfo.Iowait
}

func extractMemInfo(stats *PerformanceStats, memInfo *psproc.MemoryInfoStat) {
	stats.Rss = memInfo.RSS
	stats.Vms = memInfo.VMS
	stats.Swap = memInfo.Swap
}

func extractIoInfo(stats *PerformanceStats, ioInfo *psproc.IOCountersStat) {
	stats.ReadCount = ioInfo.ReadCount
	stats.WriteCount = ioInfo.WriteCount
	stats.ReadBytes = ioInfo.ReadBytes
	stats.WriteBytes = ioInfo.WriteBytes
}

func collectChildrenProcesses(process *psproc.Process) []*psproc.Process {
	processes := []*psproc.Process{}
	children, err := process.Children()

	if err == nil {
		processes = append(processes, children...)
		for _, child := range children {
			processes = append(processes, collectChildrenProcesses(child)...)
		}
	}

	return processes
}

func aggregateStats(stats1 *PerformanceStats, stats2 *PerformanceStats) *PerformanceStats {
	stats1.Iowait += stats2.Iowait
	stats1.ReadBytes += stats2.ReadBytes
	stats1.ReadCount += stats2.ReadCount
	stats1.Rss += stats2.Rss
	stats1.Stime += stats2.Stime
	stats1.Swap += stats2.Swap
	stats1.Utime += stats2.Utime
	stats1.Vms += stats2.Vms
	stats1.WriteBytes += stats2.WriteBytes
	stats1.WriteCount += stats2.WriteCount
	stats1.ProcessCount += stats2.ProcessCount

	if stats2.Timestamp > stats1.Timestamp {
		stats1.Timestamp = stats2.Timestamp
	}

	return stats1
}

// CollectPerformanceStats - given a PID and a pointer to psutil struct, this function returns
// an error when there is no process with the given PID
// or a map [PID] = PerformanceStats struct for the given PID and all of its children
func CollectPerformanceStats(pid int, ps *PsUtil) (map[int32]*PerformanceStats, error) {
	process, err := newProcessFunc(int32(pid))

	if err != nil {
		return nil, errors.New("Could not create process with pid " + strconv.Itoa(pid) + ": " + err.Error())
	}

	var processesStats = make(map[int32]*PerformanceStats)

	stats, err := ExtractPerformanceStats(process, ps)
	if err != nil {
		return nil, errors.New("Could not extract performance stats for pid " + strconv.Itoa(pid))
	}

	processesStats[int32(pid)] = stats

	childrenProcs := collectChildrenProcesses(process)
	for _, childProc := range childrenProcs {
		childStats, err := ExtractPerformanceStats(childProc, ps)
		if err == nil {
			processesStats[childProc.Pid] = childStats
		}
	}

	return processesStats, nil
}

// ExtractPerformanceStats reads resource consumption for the given pid
func ExtractPerformanceStats(process *psproc.Process, ps *PsUtil) (*PerformanceStats, error) {
	perfStats := new(PerformanceStats)
	perfStats.Timestamp = time.Now().Unix()

	ioStats, err := ps.getIoStats(process)
	if err != nil {
		return nil, err
	}

	memoryStat, err := ps.getMemoryStats(process)
	if err != nil {
		return nil, err
	}

	cpuStats, err := ps.getCPUTimes(process)
	if err != nil {
		return nil, err
	}

	extractIoInfo(perfStats, ioStats)
	extractMemInfo(perfStats, memoryStat)
	extractTimes(perfStats, cpuStats)
	perfStats.ProcessCount = 1

	return perfStats, nil
}

// AggregatePerformanceStatsMaps makes a union of two perf stats, assumes the second
// argument are measurements collected after the first argument
func AggregatePerformanceStatsMaps(stats1 map[int32]*PerformanceStats, stats2 map[int32]*PerformanceStats) map[int32]*PerformanceStats {
	agg := make(map[int32]*PerformanceStats)

	for pid, stats := range stats1 {
		if newerStats, ok := stats2[pid]; ok {
			agg[pid] = newerStats
		} else {
			agg[pid] = stats
		}
	}

	for pid, stats := range stats2 {
		if _, ok := agg[pid]; !ok {
			agg[pid] = stats
		}
	}

	return agg
}

// AggregatePerformanceStats - sums stats regarind a process and its children into a single struct
func AggregatePerformanceStats(stats map[int32]*PerformanceStats) *PerformanceStats {
	agg := new(PerformanceStats)

	for _, procStats := range stats {
		agg = aggregateStats(agg, procStats)
	}

	return agg
}

// RunProcessMonitoring starts online process monitoring till process ends
func RunProcessMonitoring(pid int, sim *SimulationManager, em *ExperimentManager, simulationIndex int) {
	ps := PsUtil{
		getHostInfo:    pshost.Info,
		getCPUInfo:     pscpu.Info,
		getCPUTimes:    GetCPUTimes,
		getIoStats:     GetIOStats,
		getMemoryStats: GetMemoryStats,
	}

	hostInfo, err := ExtractHostInfo(&ps)
	if err != nil {
		fmt.Printf("[SiM] Could not extract host info - %v\n", err)
		return
	}

	err = em.ReportHostInfo(simulationIndex, hostInfo)
	if err != nil {
		fmt.Printf("[SiM] An error occurred during 'ReportHostInfo' - %v\n", err)
	}

	if sim.Config.MonitoringInterval > 0 {
		pidExist, pidCheckErr := psproc.PidExists(int32(pid))
		// initialize an empty map for last stats
		lastPerformanceStats := make(map[int32]*PerformanceStats)

		for pidExist && pidCheckErr == nil {
			// this gets current stats
			currentPerformanceStats, err := CollectPerformanceStats(pid, &ps)
			if err != nil {
				fmt.Printf("[SiM] Could not extract performance statistics - %v\n", err)
				return
			}
			// aggregate last and current stats
			lastPerformanceStats = AggregatePerformanceStatsMaps(lastPerformanceStats, currentPerformanceStats)
			// sum stats from all processes into a single struct
			aggregatedPerformanceStats := AggregatePerformanceStats(lastPerformanceStats)

			// report aggregated stats
			err = em.ReportPerformanceStats(simulationIndex, aggregatedPerformanceStats)
			if err != nil {
				fmt.Printf("[SiM] An error occurred during 'ReportPerformanceStats' - %v\n", err)
			}

			time.Sleep(time.Duration(sim.Config.MonitoringInterval) * time.Second)
			pidExist, pidCheckErr = psproc.PidExists(int32(pid))
		}
	}
}
