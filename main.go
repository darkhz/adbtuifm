package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	initAPath   string
	initLPath   string
	initSelPath string
	initAuxPath string
	initSelMode ifaceMode
	initAuxMode ifaceMode
)

func main() {
	cmdAPath := kingpin.Flag("remote", "Specify the remote path to start in").
		Default("/sdcard").String()

	cmdLPath := kingpin.Flag("local", "Specify the local path to start in").
		Default("/home").String()

	kingpin.Parse()

	_, err := os.Lstat(*cmdLPath)
	if err != nil {
		fmt.Printf("adbtuifm: %s: Invalid local path\n", *cmdLPath)
		return
	}

	initSelMode = mLocal
	initSelPath, _ = filepath.Abs(*cmdLPath)

	device, err := getAdb()
	if device != nil {
		_, err := device.Stat(*cmdAPath)
		if err != nil {
			fmt.Printf("adbtuifm: %s: Invalid remote path\n", *cmdAPath)
			return
		}

		initAuxMode = mAdb
		initAuxPath = *cmdAPath
	} else {
		initAuxMode = mLocal
		initAuxPath = initSelPath
	}

	initAPath = *cmdAPath
	initLPath, _ = filepath.Abs(*cmdLPath)

	jobNum = 0
	selected = false
	openFiles = make(map[string]struct{})
	multiselection = make(map[string]ifaceMode)

	sig := make(chan os.Signal, 1)
	signal.Notify(
		sig,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)

	go func(s chan os.Signal) {
		switch <-s {
		case os.Interrupt:
			return
		default:
			stopUI()
			return
		}
	}(sig)

	setupUI()
}
