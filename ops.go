package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

type operation struct {
	id        int
	currFile  int
	totalFile int
	selindex  int
	finished  bool
	opmode    opsMode
	ctx       context.Context
	cancel    context.CancelFunc
	transfer  transferMode
	selLock   sync.Mutex
}

type ifaceMode int

const (
	mAdb ifaceMode = iota
	mLocal
)

type transferMode int

const (
	adbToAdb transferMode = iota
	adbToLocal
	localToAdb
	localToLocal
)

type opsMode int

const (
	opCopy opsMode = iota
	opMove
	opMkdir
	opRename
	opDelete
)

func (m opsMode) String() string {
	opstr := [...]string{
		"Copy",
		"Move",
		"Mkdir",
		"Rename",
		"Delete",
	}

	return opstr[m]
}

var (
	jobNum  int
	jobList []operation

	opPaths    []string
	opPathLock sync.Mutex
)

func newOperation(opmode opsMode) operation {
	transfer := localToLocal
	ctx, cancel := context.WithCancel(context.Background())

	return operation{
		id:        jobNum,
		opmode:    opmode,
		ctx:       ctx,
		cancel:    cancel,
		transfer:  transfer,
		currFile:  0,
		totalFile: 0,
	}
}

func startOperation(srcPane, dstPane *dirPane, opmode opsMode, mselect []selection) {
	var err error

	total := len(mselect)

	op := newOperation(opmode)
	jobList = append(jobList, op)

	op.opLog(opInProgress, nil)

	for sel, msel := range mselect {
		dpath := dstPane.getPath()
		src, dst := op.setNewProgress(msel.path, dpath, sel, total)

		if err = isSamePath(src, dst, opmode); err != nil {
			break
		}

		if err = addOpsPath(src, dst); err != nil {
			break
		}

		op.transfer = transfermode(opmode, msel.smode, dstPane.mode)

		switch op.transfer {
		case localToLocal:
			err = op.localOps(src, dst)

		default:
			err = op.adbOps(src, dst)
		}

		rmOpsPath(src, dst)

		if err != nil {
			break
		}
	}

	op.opLog(opDone, err)

	srcPane.ChangeDir(false, false)
	dstPane.ChangeDir(false, false)
}

func transfermode(opmode opsMode, srcMode, dstMode ifaceMode) transferMode {
	switch opmode {
	case opDelete, opRename, opMkdir:
		switch dstMode {
		case mAdb:
			return adbToAdb

		case mLocal:
			return localToLocal
		}

	default:
		switch {
		case srcMode == mLocal && dstMode == mAdb:
			return localToAdb

		case srcMode == mAdb && dstMode == mLocal:
			return adbToLocal

		case srcMode == mAdb && dstMode == mAdb:
			return adbToAdb
		}
	}

	return localToLocal
}

func isSamePath(src, dst string, opmode opsMode) error {
	switch opmode {
	case opDelete, opMkdir:
		return nil
	}

	rel, err := filepath.Rel(src, dst)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(rel, "..") {
		return fmt.Errorf("Cannot %s %s to %s", opmode.String(), src, dst)
	}

	return nil
}

func addOpsPath(src, dst string) error {
	opPathLock.Lock()
	defer opPathLock.Unlock()

	for _, path := range opPaths {
		if path == src || path == dst {
			return fmt.Errorf("Already operating on %s", path)
		}
	}

	if dst != "" {
		opPaths = append(opPaths, dst)
	}

	opPaths = append(opPaths, src)

	return nil
}

func rmOpsPath(src, dst string) {
	opPathLock.Lock()
	defer opPathLock.Unlock()

	var paths []string

	for _, path := range opPaths {
		if path == src || path == dst {
			continue
		}

		paths = append(paths, path)
	}

	opPaths = paths
}

func jobFinished(id int) {
	jobList[id].finished = true
}

func cancelOps(id int) {
	if !getUpdateLock() {
		return
	}
	defer setUpdateUnlock()

	jobList[id].cancel()
}

func cancelAllOps() {
	for id := range jobList {
		cancelOps(id)
	}
}

func clearAllOps() {
	if !getUpdateLock() {
		return
	}
	defer setUpdateUnlock()

	for id := range jobList {
		if !jobList[id].finished {
			return
		}
	}

	jobNum = 0
	jobList = nil

	setupOpsView()
}

func confirmOperation(selPane, auxPane *dirPane, opmode opsMode, mselect []selection) {
	var alert bool
	var paths []string

	doFunc := func() {
		go startOperation(selPane, auxPane, opmode, mselect)
	}

	resetFunc := func() {
		reset(auxPane, selPane)
	}

	dstpath := auxPane.getPath()
	msg := fmt.Sprintf("%s selected item(s)", opmode.String())

	switch opmode {
	case opRename, opMkdir:
		doFunc()
		return

	case opDelete:
		alert = true
		msg = fmt.Sprintf("%s from", msg)

	default:
		alert = false
		msg = fmt.Sprintf("%s to", msg)
	}

	for i := range mselect {
		paths = append(paths, mselect[i].path)
	}

	msg = fmt.Sprintf("%s %s?\n\n%s", msg, dstpath, strings.Join(paths, "\n"))

	showConfirmModal(msg, alert, doFunc, resetFunc)
}
