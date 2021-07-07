package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dolmen-go/contextio"
	adb "github.com/zach-klippenstein/goadb"
)

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

func checkAdb() bool {
	client, device := getAdb()
	if client == nil || device == nil {
		return false
	}

	return true
}

func (o *opsWork) adbTolocalOps(device *adb.Device) {
	if o.ops == opMove || o.ops == opDelete {
		o.opErr(notImplError)
		return
	}

	_, err := device.Stat(o.src)
	if adb.HasErrCode(err, adb.ErrCode(adb.FileNoExistError)) {
		o.opErr(openError)
		return
	} else if err != nil {
		o.opErr(statError)
		return
	}

	remote, err := device.OpenRead(o.src)
	if err != nil {
		o.opErr(openError)
		return
	}
	defer remote.Close()

	_, fname := filepath.Split(o.src)
	local, err := os.Create(o.dst + fname)
	if err != nil {
		o.opErr(createError)
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
		o.opErr(notImplError)
		return
	}

	local, err := os.Open(o.src)
	if err != nil {
		o.opErr(openError)
		return
	}
	defer local.Close()

	localInfo, err := os.Stat(o.src)
	if err != nil {
		o.opErr(statError)
		return
	}
	perms := localInfo.Mode().Perm()
	mtime := localInfo.ModTime()

	_, fname := filepath.Split(o.src)
	remote, err := device.OpenWrite(o.dst+fname, perms, mtime)
	if err != nil {
		o.opErr(createError)
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
		if o.pane.isDir(o.src) {
			cmd = "cp -r"
		} else {
			cmd = "cp"
		}
	case opDelete:
		if o.pane.isDir(o.src) {
			cmd = "rm -rf"
		} else {
			cmd = "rm"
		}
		param = src
	}

	cmd = cmd + param
	_, err := device.RunCommand(cmd)
	if err != nil {
		showError(unknownError, "during an ADB "+o.ops.String()+" operation")
		return
	}

	o.opLog(opDone, err)
}

func (o *opsWork) adbOps() {
	client, device := getAdb()
	if client == nil || device == nil {
		o.opErr(adbError)
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

func (p *dirPane) adbListDir(testPath string) bool {
	client, device := getAdb()
	if client == nil || device == nil {
		return false
	}

	_, err := device.Stat(testPath)
	if adb.HasErrCode(err, adb.ErrCode(adb.FileNoExistError)) {
		showError(statError, testPath)
		return false
	} else if err != nil {
		showError(unknownError, testPath)
		return false
	}

	if p.pathList != nil && !p.isDir(testPath) {
		showError(unknownError, testPath)
		return false
	}

	dent, err := device.ListDirEntries(testPath)
	if err != nil {
		showError(statError, testPath)
		return false
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
		return false
	}

	return true
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
