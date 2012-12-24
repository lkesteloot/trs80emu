package main

import (
	"log"
	"os"
	"runtime/pprof"
)

const (
	profiling       = false
	profileFilename = "trs80.prof"
)

func main() {
	if profiling {
		// When profiling don't run the web server, for some reason it causes
		// the profile file to be empty.
		profileSystem()
	} else {
		serveWebsite()
	}
}

func profileSystem() {
	vm := createVm(nil)

	f, err := os.Create(profileFilename)
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	vm.reset(true)
	for vm.clock < cpuHz*50 {
		vm.step()
	}
}
