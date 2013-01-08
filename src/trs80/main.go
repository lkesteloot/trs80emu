// Copyright 2012 Lawrence Kesteloot

package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"
)

const (
	profileFilename = "trs80.prof"
)

// Command-line flags.
var profiling = flag.Bool("profile", false, "run for a few seconds and dump profiling file")

func main() {
	flag.Parse()

	if *profiling {
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
