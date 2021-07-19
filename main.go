package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	if strings.EqualFold(*cmdMode, "ADB") {
		device, err := getAdb()
		if err != nil {
			fmt.Println("adbtuifm: No ADB device or device unauthorized")
			return
		}

		_, err = device.Stat(*cmdAPath)
		if err != nil {
			fmt.Printf("adbtuifm: %s: Invalid ADB Path", *cmdAPath)
			return
		}

		initMode = mAdb
		initPath = *cmdAPath
	} else if strings.EqualFold(*cmdMode, "Local") {
		_, err := os.Stat(*cmdLPath)
		if err != nil {
			fmt.Printf("adbtuifm: %s: Invalid Local Path!", *cmdLPath)
			return
		}

		initMode = mLocal
		initPath, _ = filepath.Abs(*cmdLPath)
	} else {
		fmt.Println("adbtuifm: Invalid Mode!")
		return
	}

	initAPath = *cmdAPath
	initLPath, _ = filepath.Abs(*cmdLPath)

	jobNum = 0
	opsLock = false

	setupUI()
}
