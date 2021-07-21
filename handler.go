package main

import (
	"path"
	"path/filepath"
)

var (
	ops     opsMode
	opPaths []string
	srcPath string
	dstPath string
	opsLock bool
)

func modeSwitchHandler(pane *dirPane) {
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

func opsHandler(selPane *dirPane, auxPane *dirPane, key rune) {
	switch key {
	case 'c', 'm', 'd':
		if opsLock {
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
			opsLock = true

			app.SetFocus(auxPane.tbl)
			auxPane.tbl.SetSelectable(true, true)
			selPane.tbl.SetSelectable(false, false)

			return
		}

		dstPath = ""
		selPane.tbl.SetSelectable(true, true)
		auxPane.tbl.SetSelectable(false, false)
	case 'p':
		if !opsLock {
			return
		}

		opsLock = false
		dstPath = filepath.Join(selPane.path, path.Base(srcPath))
	}

	showOpConfirm(ops, srcPath, dstPath, func() {
		go startOpsWork(auxPane, selPane, ops, srcPath, dstPath)
	})
}
