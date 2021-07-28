package main

import (
	"context"
	"fmt"
	"sync"
)

var (
	jobNum  int
	jobList []opsWork

	opPathLock sync.Mutex
)

func newOpsWork(pane *dirPane, ops opsMode, srcPath, dstPath string) opsWork {
	ctx, cancel := context.WithCancel(context.Background())

	return opsWork{
		id:        jobNum,
		src:       srcPath,
		dst:       dstPath,
		pane:      pane,
		ctx:       ctx,
		cancel:    cancel,
		ops:       ops,
		transfer:  localToLocal,
		finished:  false,
		currFile:  0,
		totalFile: 0,
	}
}

func startOpsWork(srcPane, dstPane *dirPane, ops opsMode, srcPath, dstPath string) {
	if checkOpPaths(srcPath, dstPath) {
		return
	}

	op := newOpsWork(dstPane, ops, srcPath, dstPath)

	jobList = append(jobList, op)

	if (srcPane.mode == mLocal) && (dstPane.mode == mAdb) {
		op.transfer = localToAdb
	} else if (srcPane.mode == mAdb) && (dstPane.mode == mLocal) {
		op.transfer = adbToLocal
	} else if (srcPane.mode == mAdb) && (dstPane.mode == mAdb) {
		op.transfer = adbToAdb
	}

	if dstPath == "" {
		if dstPane.mode == mAdb {
			op.transfer = adbToAdb
		} else {
			op.transfer = localToLocal
		}
	}

	if op.transfer != localToLocal {
		op.adbOps()
	} else {
		op.localOps()
	}

	app.QueueUpdateDraw(func() {
		srcPane.ChangeDir(false, false)
		dstPane.ChangeDir(false, false)
	})

	removeOpPaths(srcPath, dstPath)
}

func removeOpPaths(srcPath, dstPath string) {
	opPathLock.Lock()
	defer opPathLock.Unlock()

	var paths []string

	for _, path := range opPaths {
		if path == srcPath || path == dstPath {
			continue
		}

		paths = append(paths, path)
	}

	opPaths = paths
}

func checkOpPaths(srcPath, dstPath string) bool {
	opPathLock.Lock()
	defer opPathLock.Unlock()

	for _, path := range opPaths {
		if path == srcPath || path == dstPath {
			err := fmt.Errorf("Already operating on %s", path)
			showError(err, false)
			return true
		}
	}

	if dstPath != "" {
		opPaths = append(opPaths, dstPath)
	}
	opPaths = append(opPaths, srcPath)

	return false
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

func showOpConfirm(op opsMode, srcPath, dstPath string, DoFunc func()) {
	alert := false

	if dstPath != "" {
		srcPath = fmt.Sprintf("%s to %s", srcPath, dstPath)
	}

	msg := fmt.Sprintf("%s %s?", op.String(), srcPath)

	if op == opDelete {
		alert = true
	}

	showConfirmModal(msg, alert, DoFunc)
}
