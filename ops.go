package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

var (
	jobNum  int
	jobList []opsWork

	opPaths    []string
	opPathLock sync.Mutex
)

func newOpsWork(ops opsMode) opsWork {
	transfer := localToLocal
	ctx, cancel := context.WithCancel(context.Background())

	return opsWork{
		id:        jobNum,
		ctx:       ctx,
		cancel:    cancel,
		ops:       ops,
		transfer:  transfer,
		finished:  false,
		currFile:  0,
		totalFile: 0,
	}
}

func startOpsWork(srcPane, dstPane *dirPane, ops opsMode, mselect []selection, altdst []string) {
	var err error

	op := newOpsWork(ops)
	totalmsel := len(mselect)

	jobList = append(jobList, op)

	op.opLog(opInProgress, nil)

	for sel, msel := range mselect {
		dpath := dstPane.getPanePath()
		src, dst := op.updatePathProgress(msel.path, dpath, altdst, sel, totalmsel)

		if err = checkSamePaths(src, dst, ops); err != nil {
			break
		}

		if err = checkOpPaths(src, dst); err != nil {
			break
		}

		op.transfer = getTransferMode(ops, msel.smode, dstPane.mode)

		switch op.transfer {
		case localToLocal:
			err = op.localOps(src, dst)
		default:
			err = op.adbOps(src, dst)
		}

		if err != nil {
			removeOpPaths(src, dst)
			break
		}

		removeOpPaths(src, dst)
	}

	op.opLog(opDone, err)

	srcPane.ChangeDir(false, false)
	dstPane.ChangeDir(false, false)
}

func getTransferMode(ops opsMode, srcMode, dstMode ifaceMode) transferMode {
	switch ops {
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

func checkSamePaths(src, dst string, ops opsMode) error {
	switch ops {
	case opDelete, opMkdir:
		return nil
	}

	rel, err := filepath.Rel(src, dst)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(rel, "..") {
		return fmt.Errorf("Cannot %s on same paths: %s %s", ops.String(), src, dst)
	}

	return nil
}

func checkOpPaths(src, dst string) error {
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

func removeOpPaths(src, dst string) {
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

	opsView.Clear()
	setupInfoView()
}

func showOpConfirm(selPane, auxPane *dirPane, op opsMode, mselect []selection, altdst []string) {
	var alert bool
	var paths []string

	doFunc := func() {
		go startOpsWork(selPane, auxPane, op, mselect, altdst)
	}

	resetFunc := func() {
		reset(auxPane, selPane)
	}

	dstpath := auxPane.getPanePath()
	msg := fmt.Sprintf("%s selected item(s)", op.String())

	switch op {
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
