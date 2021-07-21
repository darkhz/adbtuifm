package main

import (
	"context"
	"errors"
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
	if checkOpPaths(srcPath) {
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

	op.finished = true

	app.QueueUpdateDraw(func() {
		srcPane.ChangeDir(false, false)
		dstPane.ChangeDir(false, false)
	})

	removeOpPath(srcPath)
}

func removeOpPath(opPath string) {
	opPathLock.Lock()
	defer opPathLock.Unlock()

	for i, name := range opPaths {
		if name == opPath {
			opPaths[i] = opPaths[len(opPaths)-1]
			opPaths[len(opPaths)-1] = ""
			opPaths = opPaths[:len(opPaths)-1]

			return
		}
	}
}

func checkOpPaths(opPath string) bool {
	opPathLock.Lock()
	defer opPathLock.Unlock()

	for _, name := range opPaths {
		if name == opPath {
			err := errors.New("Already operating on" + name)
			showError(err, false)
			return true
		}
	}

	opPaths = append(opPaths, opPath)
	return false
}

func cancelOps(id int) {
	jobList[id].cancel()
}

func cancelAllOps() {
	for id := range jobList {
		cancelOps(id)
	}
}

func clearAllOps() bool {
	for id := range jobList {
		if !jobList[id].finished {
			return false
		}
	}

	jobNum = 0

	return true
}

func showOpConfirm(op opsMode, srcPath, dstPath string, DoFunc func()) {
	alert := false

	msg := op.String() + " " + srcPath

	if dstPath != "" {
		msg = msg + " to " + dstPath
	}

	msg = msg + "?"

	if op == opDelete {
		alert = true
	}

	showConfirmModal(msg, alert, DoFunc)
}
