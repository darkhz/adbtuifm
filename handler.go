package main

import (
	"path/filepath"
	"sync"

	"github.com/gdamore/tcell/v2"
)

var (
	ops        opsMode
	opPaths    []string
	multiPaths []string
	opslock    bool
	selstart   bool
	opsLock    sync.Mutex
	selLock    sync.Mutex
)

func (p *dirPane) modeSwitchHandler() {
	if !p.getLock() {
		return
	}
	defer p.setUnlock()

	if selstart && p.selected {
		return
	}

	switch p.mode {
	case mAdb:
		p.mode = mLocal
		p.apath = p.path
		p.path = p.dpath
	case mLocal:
		if !checkAdb() {
			return
		}
		p.mode = mAdb
		p.dpath = p.path
		p.path = p.apath
	}

	p.ChangeDir(false, false)
}

func opsHandler(selPane, auxPane *dirPane, key rune, altdst ...string) {
	if !selPane.getLock() {
		return
	}
	defer selPane.setUnlock()

	var opstmp opsMode
	var srctmp []string

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
			multiPaths = append(multiPaths, srcpath)
		}

		switch key {
		case 'c':
			ops = opCopy
		case 'm':
			ops = opMove
		case 'd':
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

		opstmp = ops
		srctmp = multiPaths

	case 'p':
		if !getOpsLock() {
			return
		}

		selstart = false
		setOpsLock(false)

		opstmp = ops
		srctmp = multiPaths

		selPane.setPaneOpStatus(false)

	case 'M', 'R':
		var srcpath string

		switch key {
		case 'M':
			opstmp = opMkdir
			srcpath = filepath.Join(selPane.path, altdst[0])

		case 'R':
			opstmp = opRename

			selection := selPane.tbl.GetCell(selPane.row, 0).Text
			srcpath = filepath.Join(selPane.path, selection)
			checkSelected(srcpath, true)

			altdst[0] = filepath.Join(selPane.path, altdst[0])
		}

		srctmp = []string{srcpath}
	}

	if altdst == nil {
		altdst = append(altdst, "")
	}

	showOpConfirm(auxPane, selPane, opstmp, srctmp, altdst)
}

func (p *dirPane) multiSelectHandler(all bool) {
	go func() {
		if !p.getLock() {
			return
		}
		defer p.setUnlock()

		if getOpsLock() && selstart {
			return
		}

		app.QueueUpdateDraw(func() {
			selstart = true
			p.selected = true

			rows := 1
			totalrows := p.tbl.GetRowCount()

			if all {
				multiPaths = nil
				rows = totalrows
			}

			for i := 0; i < rows; i++ {
				if !all {
					i, _ = p.tbl.GetSelection()
				}

				cell := p.tbl.GetCell(i, 0)
				if cell.Text == "" {
					return
				}

				text := filepath.Join(p.path, cell.Text)

				if checkSelected(text, true) {
					p.tbl.SetCell(i, 0, cell.SetTextColor(tcell.ColorSkyblue))

					if !all {
						return
					}

					continue
				}

				selLock.Lock()
				multiPaths = append(multiPaths, text)
				selLock.Unlock()

				p.tbl.SetCell(i, 0, cell.SetTextColor(tcell.ColorOrange))

				if i+1 < totalrows && !all {
					p.tbl.Select(i+1, 0)
				}
			}
		})
	}()
}

func checkSelected(selpath string, rm bool) bool {
	selLock.Lock()
	defer selLock.Unlock()

	if !selstart {
		return false
	}

	for i, spath := range multiPaths {
		if selpath == spath {
			if rm {
				multiPaths[i] = multiPaths[len(multiPaths)-1]
				multiPaths = multiPaths[:len(multiPaths)-1]
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
