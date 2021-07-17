package main

import (
	"os"
	"strings"

	adb "github.com/zach-klippenstein/goadb"
)

func checkAdb() bool {
	client, device := getAdb()
	if client == nil || device == nil {
		return false
	}

	return true
}

func getAdb() (*adb.Adb, *adb.Device) {
	client, err := adb.NewWithConfig(adb.ServerConfig{})
	if err != nil {
		return nil, nil
	}

	device := client.Device(adb.AnyDevice())

	state, err := device.State()
	if err != nil || state != adb.StateOnline {
		if app != nil {
			showError(adbError, "")
		}
		return client, nil
	}

	return client, device
}

func isSymDir(testPath, name string) bool {
	client, device := getAdb()
	if client == nil || device == nil {
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

	client, device := getAdb()
	if client == nil || device == nil {
		o.opErr(adbError)
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
		showError(unknownError, "during an ADB "+o.ops.String()+" operation")
		return err
	}

	return nil
}

func (p *dirPane) adbListDir(testPath string, autocomplete bool) ([]string, bool) {
	var dlist []string

	client, device := getAdb()
	if client == nil || device == nil {
		return nil, false
	}

	_, err := device.Stat(testPath)
	if adb.HasErrCode(err, adb.ErrCode(adb.FileNoExistError)) {
		if !autocomplete {
			showError(statError, testPath)
		}
		return nil, false
	} else if err != nil {
		if !autocomplete {
			showError(unknownError, testPath)
		}
		return nil, false
	}

	if !autocomplete && p.pathList != nil && !p.isDir(testPath) {
		if !autocomplete {
			showError(unknownError, testPath)
		}
		return nil, false
	}

	dent, err := device.ListDirEntries(testPath)
	if err != nil {
		if !autocomplete {
			showError(statError, testPath)
		}
		return nil, false
	}

	for dent.Next() {
		ent := dent.Entry()
		name := ent.Name

		if name == ".." || name == "." {
			continue
		}

		if setHidden && strings.HasPrefix(name, ".") {
			continue
		}

		if ent.Mode&os.ModeDir != 0 {
			if !autocomplete {
				ent.Name = name + "/"
			} else {
				dlist = append(dlist, testPath+name)
				continue
			}
		}

		if ent.Mode&os.ModeDir == 0 && autocomplete {
			continue
		}

		if !autocomplete {
			p.pathList = append(p.pathList, ent)
		}
	}
	if dent.Err() != nil {
		return nil, false
	}

	return dlist, true
}
