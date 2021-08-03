package main

import (
	"path/filepath"
	"sync"
)

var (
	ops      opsMode
	opPaths  []string
	srcPaths []string
	opslock  bool
	selstart bool
	opsLock  sync.Mutex
	selLock  sync.Mutex
)

func modeSwitchHandler(pane *dirPane) {
	if !pane.getLock() {
		return
	}
	defer pane.setUnlock()

	if selstart && pane.selected {
		return
	}

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

	selPane.updateRow(false)

	switch key {
	case 'c', 'm', 'd':
		if getOpsLock() {
			return
		}

		if !selstart {
			selection := selPane.tbl.GetCell(selPane.row, 0).Text
			if selection == "" {
				setOpsLock(false)
				selPane.setPaneOpStatus(false)

				return
			}

			srcpath := filepath.Join(selPane.path, selection)
			srcPaths = append(srcPaths, srcpath)
		}

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

		selPane.tbl.SetSelectable(true, true)
		auxPane.tbl.SetSelectable(false, false)
	case 'p':
		if !getOpsLock() {
			return
		}

		selstart = false
		setOpsLock(false)

		selPane.setPaneOpStatus(false)
	}

	showOpConfirm(ops, auxPane, selPane, srcPaths, func() {
		go startOpsWork(auxPane, selPane, ops, srcPaths)
	})
}

func checkSelected(selpath string, rm bool) bool {
	selLock.Lock()
	defer selLock.Unlock()

	if !selstart {
		return false
	}

	for i, spath := range srcPaths {
		if selpath == spath {
			if rm {
				srcPaths[i] = srcPaths[len(srcPaths)-1]
				srcPaths = srcPaths[:len(srcPaths)-1]
			}

			return true
		}
	}

	return false
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
