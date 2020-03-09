package feedservice

import (
	"fmt"
	"runtime"
	"sync"

	log "github.com/sirupsen/logrus"
)

const toMeg uint64 = 1048576

// PipelineError let's you set IsNonCritical in case the pipeline should not be stopped
type PipelineError struct {
	IsNonCritical bool
	Message       error
	mux           *sync.Mutex // points to the PipeLineErrors mux
}

// Error Implements the error interface
func (e PipelineError) Error() string {
	return fmt.Sprintf("Error:%s, IsNonCritical: %t\n", e.Message, e.IsNonCritical)
}

// PipelineErrors collects errors and stops when they are critical
type PipelineErrors struct {
	Errors       []error
	Critical     bool
	mux          *sync.Mutex // points to the pipeline mux
	production   bool
	maxMemoryUse *uint64
	mem          runtime.MemStats
}

func NewPE(mux *sync.Mutex, productionFlag bool) PipelineErrors {
	return PipelineErrors{
		mux:          mux,
		production:   productionFlag,
		maxMemoryUse: new(uint64),
	}
}

func (pe *PipelineErrors) GetMaxMemory() uint64 {
	return *pe.maxMemoryUse
}

// Log appends your error to the PipleErros Log
// and updates the overall state of the pipeline to critical or not
func (pe *PipelineErrors) Log(e error, stageName string) {
	defer memLog(stageName, pe.mem, pe.maxMemoryUse)

	if e == nil {
		return
	}

	pe.mux.Lock()
	defer pe.mux.Unlock()

	// Try to assert that the new error is a PipelineError
	// Convert explicitly if not
	// For standard errors we assume that they are always critical
	if err, ok := e.(PipelineError); ok {
		err.mux = pe.mux
		err.Message = fmt.Errorf("%s - %s", stageName, err.Message)
		pe.Errors = append(pe.Errors, err)

		if err.IsNonCritical {
			log.Warnf("%v", err)
			return
		}
	} else {
		pe.Errors = append(pe.Errors, PipelineError{
			mux:     pe.mux,
			Message: fmt.Errorf("%s - %v", stageName, e),
		})
	}

	log.WithFields(log.Fields{
		"critical error": e,
		"other errors":   pe.Errors,
	}).Fatal("Pipeline Stopped")
}

func (pe PipelineErrors) Error() string {
	pe.mux.Lock()
	defer pe.mux.Unlock()

	var output string

	for _, logError := range pe.Errors {
		output += logError.Error() + "\n"
	}

	return output
}

func memLog(message string, mem runtime.MemStats, maxMemory *uint64) {
	runtime.ReadMemStats(&mem)

	log.WithFields(log.Fields{
		"Mem Allocated": mem.Alloc / toMeg,
		"HeapAlloc":     mem.HeapAlloc / toMeg,
		"System Memory": mem.Sys / toMeg,
		"Go Routines":   runtime.NumGoroutine(),
		"Num GC":        mem.NumGC,
	}).Info(message)

	if mem.Alloc/toMeg > *maxMemory {
		*maxMemory = mem.Alloc / toMeg
	}
}
