package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

var (
	jobNum  int
	jobList []opsWork

	opPathLock sync.Mutex
)

func newOpsWork(ops opsMode) opsWork {
	ctx, cancel := context.WithCancel(context.Background())

	return opsWork{
		id:        jobNum,
		ctx:       ctx,
		cancel:    cancel,
		ops:       ops,
		transfer:  localToLocal,
		finished:  false,
		currFile:  0,
		totalFile: 0,
	}
}

func startOpsWork(srcPane, dstPane *dirPane, ops opsMode, srcs []string) {
	var err error

	tsrcs := len(srcs)
	op := newOpsWork(ops)

	jobList = append(jobList, op)

	if op.ops == opDelete {
		switch dstPane.mode {
		case mAdb:
			op.transfer = adbToAdb
		case mLocal:
			op.transfer = localToLocal
		}

		goto OPSTART
	}

	switch {
	case srcPane.mode == mLocal && dstPane.mode == mAdb:
		op.transfer = localToAdb
	case srcPane.mode == mAdb && dstPane.mode == mLocal:
		op.transfer = adbToLocal
	case srcPane.mode == mAdb && dstPane.mode == mAdb:
		op.transfer = adbToAdb
	case srcPane.mode == mLocal && dstPane.mode == mLocal:
		op.transfer = localToLocal
	}

OPSTART:
	op.opLog(opInProgress, nil)

	for csrc, src := range srcs {
		dst := op.updatePathProgress(src, dstPane.getPanePath(), csrc, tsrcs)

		if err = checkOpPaths(src, dst); err != nil {
			break
		}

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

	dstPane.ChangeDir(false, false)
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

func checkOpPaths(src, dst string) error {
	opPathLock.Lock()
	defer opPathLock.Unlock()

	for _, path := range opPaths {
		if path == src || path == dst {
			err := fmt.Errorf("Already operating on %s", path)
			showError(err, false)
			return err
		}
	}

	if dst != "" {
		opPaths = append(opPaths, dst)
	}
	opPaths = append(opPaths, src)

	return nil
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

func showOpConfirm(op opsMode, selPane, auxPane *dirPane, paths []string, doFunc func()) {
	alert := true
	dstpath := auxPane.getPanePath()

	msg := fmt.Sprintf("%s selected item(s)", op.String())

	if op != opDelete {
		alert = false
		msg = fmt.Sprintf("%s to", msg)
	} else {
		msg = fmt.Sprintf("%s from", msg)
	}

	msg = fmt.Sprintf("%s %s?\n\n%s", msg, dstpath, strings.Join(paths, "\n"))

	showConfirmModal(msg, alert, doFunc, func() { reset(auxPane, selPane) })
}
