package main

import (
	"path"
	"path/filepath"
	"sync"
)

var (
	ops     opsMode
	opPaths []string
	srcPath string
	dstPath string
	opslock bool
	opsLock sync.Mutex
)

func modeSwitchHandler(pane *dirPane) {
	if pane.getPending() {
		return
	}

	if !pane.getLock() {
		return
	}
	defer pane.setUnlock()

	switch pane.mode {
	case mAdb:
		pane.mode = mLocal
		pane.apath = pane.path
		pane.path = pane.dpath
	case mLocal:
		if !checkAdb() {
			return
		}
		pane.mode = mAdb
		pane.dpath = pane.path
		pane.path = pane.apath
	}

	pane.ChangeDir(false, false)
}

func opsHandler(selPane, auxPane *dirPane, key rune) {
	if !selPane.getLock() {
		return
	}
	defer selPane.setUnlock()

	prevPane = selPane
	selPane.updateRow(false)

	switch key {
	case 'c', 'm', 'd':
		if getOpsLock() {
			return
		}

		selection := selPane.tbl.GetCell(selPane.row, 0).Text
		srcPath = filepath.Join(selPane.path, selection)

		if key == 'm' {
			ops = opMove
		} else if key == 'c' {
			ops = opCopy
		} else {
			ops = opDelete
		}

		if ops == opMove || ops == opCopy {
			setOpsLock(true)

			app.SetFocus(auxPane.tbl)
			auxPane.tbl.SetSelectable(true, true)
			selPane.tbl.SetSelectable(false, false)

			auxPane.setPaneOpStatus(true)

			return
		}

		dstPath = ""
		selPane.tbl.SetSelectable(true, true)
		auxPane.tbl.SetSelectable(false, false)
	case 'p':
		if !getOpsLock() {
			return
		}

		setOpsLock(false)
		dstPath = filepath.Join(selPane.path, path.Base(srcPath))

		selPane.setPaneOpStatus(false)
	}

	showOpConfirm(ops, srcPath, dstPath, func() {
		go startOpsWork(auxPane, selPane, ops, srcPath, dstPath)
	})
}

func getOpsLock() bool {
	opsLock.Lock()
	defer opsLock.Unlock()

	return opslock
}

func setOpsLock(setlock bool) {
	opsLock.Lock()
	defer opsLock.Unlock()

	opslock = setlock
}
