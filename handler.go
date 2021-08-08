package main

import (
	"path/filepath"
	"sync"

	"github.com/gdamore/tcell/v2"
)

var (
	selected   bool
	selectLock sync.Mutex
	multiPaths []selection
)

func (p *dirPane) modeSwitchHandler() {
	if !p.getLock() {
		return
	}
	defer p.setUnlock()

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
	var srctmp []selection

	switch key {
	case 'p', 'm', 'd':
		if multiPaths == nil {
			return
		}

		switch key {
		case 'p':
			opstmp = opCopy
		case 'm':
			opstmp = opMove
		case 'd':
			opstmp = opDelete
		}

		srctmp = multiPaths

	case 'M', 'R':
		if altdst == nil {
			return
		}

		var srcpath string

		switch key {
		case 'M':
			opstmp = opMkdir
			srcpath = filepath.Join(selPane.path, altdst[0])

		case 'R':
			opstmp = opRename

			selPane.updateRow(false)
			selection := selPane.tbl.GetCell(selPane.row, 0).Text
			srcpath = filepath.Join(selPane.path, selection)
			checkSelected(srcpath, true)

			altdst[0] = filepath.Join(selPane.path, altdst[0])
		}

		srctmp = []selection{{srcpath, selPane.mode}}
	}

	showOpConfirm(auxPane, selPane, opstmp, srctmp, altdst)
}

func multiSelectHandler(selPane *dirPane, all bool, totalrows int) {
	go func() {
		if !selPane.getLock() {
			return
		}
		defer selPane.setUnlock()

		app.QueueUpdateDraw(func() {
			rows := 1
			selected = true

			if all {
				multiPaths = nil
				rows = totalrows
			}

			for i := 0; i < rows; i++ {
				if !all {
					i, _ = selPane.tbl.GetSelection()
				}

				cell := selPane.tbl.GetCell(i, 0)
				if cell.Text == "" {
					return
				}

				text := filepath.Join(selPane.path, cell.Text)

				if checkSelected(text, true) {
					selPane.tbl.SetCell(i, 0, cell.SetTextColor(tcell.ColorSkyblue))

					if !all {
						return
					}

					continue
				}

				selectLock.Lock()
				multiPaths = append(multiPaths, selection{text, selPane.mode})
				selectLock.Unlock()

				selPane.tbl.SetCell(i, 0, cell.SetTextColor(tcell.ColorOrange))

				if i+1 < totalrows && !all {
					selPane.tbl.Select(i+1, 0)
				}
			}
		})
	}()
}

func checkSelected(selpath string, rm bool) bool {
	selectLock.Lock()
	defer selectLock.Unlock()

	if !selected {
		return false
	}

	for i, spath := range multiPaths {
		if selpath == spath.path {
			if rm {
				multiPaths[i] = multiPaths[len(multiPaths)-1]
				multiPaths = multiPaths[:len(multiPaths)-1]
			}

			return true
		}
	}

	return false
}
