package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dolmen-go/contextio"
	"github.com/zach-klippenstein/goadb"
)

func getAdb() (*adb.Adb, *adb.Device) {
	client, err := adb.NewWithConfig(adb.ServerConfig{})
	if err != nil {
		return nil, nil
	}

	device := client.Device(adb.AnyDevice())

	state, err := device.State()
	if err != nil || state.String() != "StateOnline" {
		if app != nil {
			showError(errors.New("Could not find ADB-connected device"), "")
		}
		return client, nil
	}

	return client, device
}

func checkAdb() bool {
	client, device := getAdb()
	if client == nil || device == nil {
		return false
	}

	return true
}

func (o *opsWork) adbTolocalOps(device *adb.Device) {
	if o.ops == opMove || o.ops == opDelete {
		o.opLog(notImplError, nil)
		return
	}

	_, err := device.Stat(o.src)
	if adb.HasErrCode(err, adb.ErrCode(adb.FileNoExistError)) {
		o.opLog(openError, err)
		return
	} else if err != nil {
		o.opLog(statError, err)
		return
	}

	remote, err := device.OpenRead(o.src)
	if err != nil {
		o.opLog(openError, err)
		return
	}
	defer remote.Close()

	_, fname := filepath.Split(o.src)
	local, err := os.Create(o.dst + fname)
	if err != nil {
		o.opLog(createError, err)
		return
	}
	defer local.Close()

	o.opLog(opInProgress, nil)

	cioOut := contextio.NewWriter(o.ctx, local)
	_, err = io.Copy(cioOut, remote)

	o.opLog(opDone, err)
}

func (o *opsWork) localToadbOps(device *adb.Device) {
	if o.ops == opMove || o.ops == opDelete {
		o.opLog(notImplError, nil)
		return
	}

	local, err := os.Open(o.src)
	if err != nil {
		o.opLog(openError, err)
		return
	}
	defer local.Close()

	localInfo, err := os.Stat(o.src)
	if err != nil {
		o.opLog(statError, err)
		return
	}
	perms := localInfo.Mode().Perm()
	mtime := localInfo.ModTime()

	_, fname := filepath.Split(o.src)
	remote, err := device.OpenWrite(o.dst+fname, perms, mtime)
	if err != nil {
		o.opLog(createError, err)
		return
	}
	defer remote.Close()

	o.opLog(opInProgress, nil)

	cioIn := contextio.NewReader(o.ctx, local)
	_, err = io.Copy(remote, cioIn)

	o.opLog(opDone, err)
}

func (o *opsWork) adbToadbOps(device *adb.Device) {
	var cmd string

	src := " " + "'" + o.src + "'"
	dst := " " + "'" + o.dst + "'"

	param := src + dst
	o.opLog(opInProgress, nil)

	switch o.ops {
	case opMove:
		cmd = "mv"
	case opCopy:
		if isDir(o.pane, o.src) {
			cmd = "cp -r"
		} else {
			cmd = "cp"
		}
	case opDelete:
		if isDir(o.pane, o.src) {
			cmd = "rm -rf"
		} else {
			cmd = "rm"
		}
		param = src
	}

	cmd = cmd + param
	_, err := device.RunCommand(cmd)
	if err != nil {
		return
	}

	o.opLog(opDone, err)
}

func (o *opsWork) adbOps() {
	client, device := getAdb()
	if client == nil || device == nil {
		o.opLog(adbError, nil)
		return
	}

	switch o.transfer {
	case adbToAdb:
		o.adbToadbOps(device)
	case localToAdb:
		o.localToadbOps(device)
	case adbToLocal:
		o.adbTolocalOps(device)
	}
}

func (p *dirPane) adbListDir(testPath string) {
	client, device := getAdb()
	if client == nil || device == nil {
		return
	}

	_, err := device.Stat(testPath)
	if adb.HasErrCode(err, adb.ErrCode(adb.FileNoExistError)) {
		showError(err, testPath)
		return
	} else if err != nil {
		showError(err, testPath)
		return
	}

	if p.pathList != nil && !isDir(p, testPath) {
		return
	}

	dent, err := device.ListDirEntries(testPath)
	if err != nil {
		showError(err, testPath)
		return
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
			ent.Name = name + "/"
		}

		p.pathList = append(p.pathList, ent)
	}
	if dent.Err() != nil {
		return
	}
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
