package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	adb "github.com/zach-klippenstein/goadb"
)

func checkAdb() bool {
	_, err := getAdb()
	if err != nil {
		showErrorModal(err.Error())
		return false
	}

	return true
}

func getAdb() (*adb.Device, error) {
	client, err := adb.NewWithConfig(adb.ServerConfig{})
	if err != nil {
		return nil, fmt.Errorf("ADB client not found")
	}

	device := client.Device(adb.AnyDevice())

	state, err := device.State()
	if err != nil || state != adb.StateOnline {
		return nil, fmt.Errorf("ADB device not found")
	}

	return device, nil
}

func isSymDir(testPath, name string) bool {
	device, err := getAdb()
	if err != nil {
		return false
	}

	cmd := fmt.Sprintf("ls -pd $(readlink -f '%s%s')", testPath, name)
	out, err := device.RunCommand(cmd)
	if err != nil {
		return false
	}

	if !strings.HasSuffix(strings.TrimSpace(out), "/") {
		return false
	}

	return true
}

func (o *opsWork) adbOps(src, dst string) error {
	var err error

	device, err := getAdb()
	if err != nil {
		showError(err, false)
		return err
	}

	switch o.transfer {
	case adbToAdb:
		err = o.execAdbOps(device)
	case localToAdb:
		err = o.pushRecursive(src, dst, device)
	case adbToLocal:
		err = o.pullRecursive(src, dst, device)
	}

	return err
}

func (o *opsWork) execAdbOps(device *adb.Device) error {
	var cmd string

	src := fmt.Sprintf(" '%s'", o.src)
	dst := fmt.Sprintf(" '%s'", o.dst)

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
		showError(err, autocomplete)
		return nil, false
	}

	_, err = device.Stat(testPath)
	if err != nil {
		showError(err, autocomplete)
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

		if p.getHidden() && strings.HasPrefix(name, ".") {
			continue
		}

		if ent.Mode&os.ModeDir != 0 {
			dlist = append(dlist, path.Join(testPath, name))
			ent.Name = name + "/"
		}

		if autocomplete {
			continue
		}

		p.pathList = append(p.pathList, ent)
	}
	if dent.Err() != nil {
		return nil, false
	}

	return dlist, true
}
