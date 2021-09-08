package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	adb "github.com/zach-klippenstein/goadb"
)

func checkAdb() bool {
	_, err := getAdb()
	if err != nil {
		showErrorMsg(err, false)
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

func isAdbSymDir(testPath, name string) bool {
	device, err := getAdb()
	if err != nil {
		return false
	}

	cmd := fmt.Sprintf("ls -pd %s%s/", testPath, name)
	out, err := device.RunCommand(cmd)
	if err != nil {
		return false
	}

	if !strings.HasSuffix(strings.TrimSpace(out), "//") {
		return false
	}

	return true
}

func (o *operation) adbOps(src, dst string) error {
	var err error

	device, err := getAdb()
	if err != nil {
		return err
	}

	switch o.transfer {
	case adbToAdb:
		err = o.execAdbCmd(src, dst, device)

	case localToAdb:
		err = o.pushRecursive(src, dst, device)

	case adbToLocal:
		err = o.pullRecursive(src, dst, device)
	}

	return err
}

func (o *operation) execAdbCmd(src, dst string, device *adb.Device) error {
	var cmd string

	srcfmt := fmt.Sprintf(" '%s'", src)
	dstfmt := fmt.Sprintf(" '%s'", dst)

	param := srcfmt + dstfmt

	switch o.opmode {
	case opMkdir:
		cmd = "mkdir"
		param = srcfmt

	default:
		stat, err := device.Stat(src)
		if err != nil {
			return err
		}

		switch o.opmode {
		case opRename:
			_, err := device.Stat(dst)
			if err == nil {
				return fmt.Errorf("rename %s %s: file exists", src, dst)
			}

			fallthrough

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

			param = srcfmt
		}
	}

	cmd = cmd + param
	out, err := exec.CommandContext(o.ctx, "adb", "shell", cmd).Output()

	if err != nil {
		if err.Error() == "signal: killed" {
			return error(context.Canceled)
		}

		return err
	}

	if string(out) != "" {
		return fmt.Errorf(string(out))
	}

	return nil
}

func (p *dirPane) adbListDir(testPath string, autocomplete bool) ([]string, bool) {
	var dlist []string

	device, err := getAdb()
	if err != nil {
		showErrorMsg(err, autocomplete)
		return nil, false
	}

	_, err = device.Stat(testPath)
	if err != nil {
		showErrorMsg(err, autocomplete)
		return nil, false
	}

	dent, err := device.ListDirEntries(testPath)
	if err != nil {
		showErrorMsg(err, autocomplete)
		return nil, false
	}

	p.pathList = nil

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
			dlist = append(dlist, filepath.Join(testPath, name))
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
