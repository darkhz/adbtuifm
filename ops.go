package main

import (
	"context"
	"sync"
)

var (
	jobNum  int
	jobList []opsWork

	opPathLock sync.Mutex
)

func newOpsWork(pane *dirPane, ops opsMode, copyPath, pastePath string) opsWork {
	ctx, cancel := context.WithCancel(context.Background())

	return opsWork{
		id:        jobNum,
		src:       copyPath,
		dst:       pastePath,
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

func startOpsWork(cpPane, psPane *dirPane, ops opsMode, cpPath, psPath string) {
	if checkOpPaths(cpPath) {
		return
	}

	op := newOpsWork(psPane, ops, cpPath, psPath)

	jobList = append(jobList, op)

	if (cpPane.mode == mLocal) && (psPane.mode == mAdb) {
		op.transfer = localToAdb
	} else if (cpPane.mode == mAdb) && (psPane.mode == mLocal) {
		op.transfer = adbToLocal
	} else if (cpPane.mode == mAdb) && (psPane.mode == mAdb) {
		op.transfer = adbToAdb
	}

	if psPath == "" {
		if psPane.mode == mAdb {
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
		cpPane.ChangeDir(false, false)
		psPane.ChangeDir(false, false)
	})

	removeOpPath(cpPath)
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
			showError(openError, "Operating on "+opPath)
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

func showOpConfirm(op opsMode, copyPath, pastePath string, DoFunc func()) {
	msg := op.String() + " " + copyPath

	if pastePath != "" {
		msg = msg + " to " + pastePath
	}

	msg = msg + "?"

	showConfirmModal(msg, DoFunc)
}
