package main

import (
	"errors"
	"os"
	"strings"

	adb "github.com/zach-klippenstein/goadb"
)

func checkAdb() bool {
	_, err := getAdb()
	if err != nil {
		showError(err, false)
		return false
	}

	return true
}

func getAdb() (*adb.Device, error) {
	clientNotFound := errors.New("ADB client not found")
	deviceNotFound := errors.New("ADB device not found")

	client, err := adb.NewWithConfig(adb.ServerConfig{})
	if err != nil {
		return nil, clientNotFound
	}

	device := client.Device(adb.AnyDevice())

	state, err := device.State()
	if err != nil || state != adb.StateOnline {
		return nil, deviceNotFound
	}

	return device, nil
}

func isSymDir(testPath, name string) bool {
	device, err := getAdb()
	if err != nil {
		return false
	}

	cmd := "ls -pd $(readlink -f " + testPath + name + ")"
	out, err := device.RunCommand(cmd)
	if err != nil {
		return false
	}

	if !strings.HasSuffix(strings.TrimSpace(out), "/") {
		return false
	}

	return true
}

func (o *opsWork) adbOps() {
	var err error

	device, err := getAdb()
	if err != nil {
		o.opErr(err)
		return
	}

	o.opLog(opInProgress, nil)

	switch o.transfer {
	case adbToAdb:
		err = o.execAdbOps(device)
	case localToAdb:
		err = o.pushRecursive(o.src, o.dst, device)
	case adbToLocal:
		err = o.pullRecursive(o.src, o.dst, device)
	}

	o.opLog(opDone, err)
}

func (o *opsWork) execAdbOps(device *adb.Device) error {
	var cmd string

	src := " " + "'" + o.src + "'"
	dst := " " + "'" + o.dst + "'"

	param := src + dst

	stat, err := device.Stat(o.src)
	if err != nil {
		return err
	}

	switch o.ops {
	case opMove:
		cmd = "mv"
	case opCopy:
		if stat.Mode.IsDir() {
			cmd = "cp -r"
		} else {
			cmd = "cp"
		}
	case opDelete:
		if stat.Mode.IsDir() {
			cmd = "rm -rf"
		} else {
			cmd = "rm"
		}
		param = src
	}

	cmd = cmd + param
	_, err = device.RunCommand(cmd)
	if err != nil {
		return err
	}

	return nil
}

func (p *dirPane) adbListDir(testPath string, autocomplete bool) ([]string, bool) {
	var dlist []string

	device, err := getAdb()
	if err != nil {
		return nil, false
	}

	_, err = device.Stat(testPath)
	if err != nil {
		showError(err, autocomplete)
		return nil, false
	}

	if !autocomplete && p.pathList != nil && !p.isDir(testPath) {
		return nil, false
	}

	dent, err := device.ListDirEntries(testPath)
	if err != nil {
		showError(err, autocomplete)
		return nil, false
	}

	for dent.Next() {
		ent := dent.Entry()
		name := ent.Name

		if name == ".." || name == "." {
			continue
		}

		if p.hidden && strings.HasPrefix(name, ".") {
			continue
		}

		if ent.Mode&os.ModeDir != 0 {
			if autocomplete {
				dlist = append(dlist, testPath+name)
				continue
			}
			ent.Name = name + "/"
		}

		p.pathList = append(p.pathList, ent)
	}
	if dent.Err() != nil {
		return nil, false
	}

	return dlist, true
}
