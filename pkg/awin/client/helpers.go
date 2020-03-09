package awinclient

import (
	"fmt"
	"runtime"
	"strconv"

	log "github.com/sirupsen/logrus"
)

var (
	maxMemory uint64
	mem       runtime.MemStats
)

func progressBar(title string, completed, total int) {
	progress := float64(completed) / float64(total) * 100.0
	s := ("[")
	for pct := 0.0; pct <= 100.0; pct += 10.0 {
		if pct <= progress {
			s += "#"
		} else {
			s += "-"
		}
	}
	s += fmt.Sprintf("] %s%% completed", strconv.FormatFloat(progress, 'f', 2, 64))

	log.WithField("Progress", s).Info(title)
}

func memLog(message string, mem runtime.MemStats, maxMemory *uint64) {
	runtime.ReadMemStats(&mem)

	log.WithFields(log.Fields{
		"Source":        "Awin",
		"Mem Allocated": mem.Alloc / toMeg,
		"HeapAlloc":     mem.HeapAlloc / toMeg,
		"System Memory": mem.Sys / toMeg,
		"Go Routines":   runtime.NumGoroutine(),
		"Num GC":        mem.NumGC,
	}).Debugln(message)

	if mem.Alloc/toMeg > *maxMemory {
		*maxMemory = mem.Alloc / toMeg
	}
}
