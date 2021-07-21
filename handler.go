package main

import (
	"path"
	"path/filepath"
)

var (
	ops       opsMode
	opPaths   []string
	srcPath  string
	dstPath string
	opsLock   bool
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
	row, _ := selPane.tbl.GetSelection()

	switch key {
	case 'c', 'm':
		if opsLock {
			return
		}

		srcPath = selPane.path + selPane.tbl.GetCell(row, 0).Text

		if key == 'm' {
			ops = opMove
		} else {
			ops = opCopy
		}

		auxPane.tbl.SetSelectable(true, true)
		selPane.tbl.SetSelectable(false, false)
		app.SetFocus(auxPane.tbl)

		opsLock = true
		return
	case 'p':
		if !opsLock {
			return
		}

		dstPath = filepath.Join(selPane.path, path.Base(srcPath))
		opsLock = false
	case 'd':
		if opsLock {
			return
		}

		auxPane.tbl.SetSelectable(false, false)
		selPane.tbl.SetSelectable(true, true)

		srcPath = selPane.path + selPane.tbl.GetCell(row, 0).Text
		dstPath = ""
		ops = opDelete
	}

	showOpConfirm(ops, srcPath, dstPath, func() {
		go startOpsWork(auxPane, selPane, ops, srcPath, dstPath)
	})
}
