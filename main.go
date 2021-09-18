package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	initPath  string
	initAPath string
	initLPath string
	initMode  ifaceMode
)

func main() {
	cmdMode := kingpin.Flag("mode", "Specify which mode to start in, ADB or Local").
		Default("Local").String()

	cmdAPath := kingpin.Flag("remote", "Specify the remote path to start in").
		Default("/sdcard").String()

	cmdLPath := kingpin.Flag("local", "Specify the local path to start in").
		Default("/home").String()

	kingpin.Parse()

	switch {
	case strings.EqualFold(*cmdMode, "ADB"):
		device, err := getAdb()
		if err != nil {
			fmt.Println("adbtuifm: No ADB device or device unauthorized")
			return
		}

		_, err = device.Stat(*cmdAPath)
		if err != nil {
			fmt.Printf("adbtuifm: %s: Invalid ADB Path\n", *cmdAPath)
			return
		}

		initMode = mAdb
		initPath = *cmdAPath

	case strings.EqualFold(*cmdMode, "Local"):
		_, err := os.Stat(*cmdLPath)
		if err != nil {
			fmt.Printf("adbtuifm: %s: Invalid Local Path!\n", *cmdLPath)
			return
		}

		initMode = mLocal
		initPath, _ = filepath.Abs(*cmdLPath)

	default:
		fmt.Println("adbtuifm: Invalid Mode!")
		return
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
