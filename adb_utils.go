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

	fname := filepath.Base(o.src)

	o.opLog(opInProgress, nil)

	stat, err := device.Stat(o.src)
	if adb.HasErrCode(err, adb.ErrCode(adb.FileNoExistError)) {
		o.opErr(openError)
		return
	} else if err != nil {
		o.opErr(statError)
		return
	}

	if stat.Mode.IsDir() {
		d := filepath.Join(o.dst, fname)

		err = o.pullRecursive(o.src, d, device)
		o.opLog(opDone, err)

		return
	}

	remote, err := device.OpenRead(o.src)
	if err != nil {
		o.opErr(openError)
		return
	}
	defer remote.Close()

	local, err := os.Create(filepath.Join(o.dst, fname))
	if err != nil {
		o.opErr(createError)
		return
	}
	defer local.Close()

	cioOut := contextio.NewWriter(o.ctx, local)
	_, err = io.Copy(cioOut, remote)

	o.opLog(opDone, err)
}

func (o *opsWork) localToadbOps(device *adb.Device) {
	if o.ops == opMove || o.ops == opDelete {
		o.opErr(notImplError)
		return
	}

	fname := filepath.Base(o.src)

	o.opLog(opInProgress, nil)

	localInfo, err := os.Stat(o.src)
	if err != nil {
		o.opErr(statError)
		return
	}

	if localInfo.Mode().IsDir() {
		d := filepath.Join(o.dst, fname)

		err = o.pushRecursive(o.src, d, device)
		o.opLog(opDone, err)

		return
	}

	local, err := os.Open(o.src)
	if err != nil {
		o.opErr(openError)
		return
	}
	defer local.Close()

	perms := localInfo.Mode().Perm()
	mtime := localInfo.ModTime()

	remote, err := device.OpenWrite(filepath.Join(o.dst, fname), perms, mtime)
	if err != nil {
		o.opErr(createError)
		return
	}
	defer remote.Close()

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

	stat, err := device.Stat(o.src)
	if err != nil {
		return
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
