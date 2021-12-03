package server

import (
	"math"
	"runtime"
	"time"
)

var start = time.Now()

const MB float64 = 1.0 * 1024 * 1024

type HealthStats struct {
	Uptime               int64   `json:"uptime"`
	AllocatedMemory      float64 `json:"allocated_memory"`
	TotalAllocatedMemory float64 `json:"total_allocated_memory"`
	Goroutines           int     `json:"goroutines"`
	GCCycles             uint32  `json:"gc_cycles"`
	NumberOfCPUs         int     `json:"number_of_cpus"`
	HeapSys              float64 `json:"heap_sys"`
	HeapAllocated        float64 `json:"heap_allocated"`
	ObjectsInUse         uint64  `json:"objects_in_use"`
	OSMemoryObtained     float64 `json:"os_memory_obtained"`
}

func GetHealthStats() *HealthStats {
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)

	return &HealthStats{
		Uptime:               GetUptime(),
		AllocatedMemory:      toMegaBytes(mem.Alloc),
		TotalAllocatedMemory: toMegaBytes(mem.TotalAlloc),
		Goroutines:           runtime.NumGoroutine(),
		NumberOfCPUs:         runtime.NumCPU(),
		GCCycles:             mem.NumGC,
		HeapSys:              toMegaBytes(mem.HeapSys),
		HeapAllocated:        toMegaBytes(mem.HeapAlloc),
		ObjectsInUse:         mem.Mallocs - mem.Frees,
		OSMemoryObtained:     toMegaBytes(mem.Sys),
	}
}

func GetUptime() int64 {
	return time.Now().Unix() - start.Unix()
}

func toMegaBytes(bytes uint64) float64 {
	return toFixed(float64(bytes)/MB, 2)
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
