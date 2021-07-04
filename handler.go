package main

var (
	ops       opsMode
	opPaths   []string
	copyPath  string
	pastePath string
	opsLock   bool = false
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

		copyPath = selPane.path + selPane.tbl.GetCell(row, 0).Text

		if key == 'm' {
			ops = opMove
		} else {
			ops = opCopy
		}

		app.SetFocus(auxPane.tbl)
		opsLock = true
		return
	case 'p':
		if !opsLock {
			return
		}

		pastePath = selPane.path
		opsLock = false
	case 'd':
		if opsLock {
			return
		}

		copyPath = selPane.path + selPane.tbl.GetCell(row, 0).Text
		pastePath = ""
		ops = opDelete
	}

	showOpConfirm(ops, copyPath, pastePath, func() {
		go startOpsWork(auxPane, selPane, ops, copyPath, pastePath)
	})
}
