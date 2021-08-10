package main

import (
	"path/filepath"
	"sync"

	"github.com/gdamore/tcell/v2"
)

type selection struct {
	path  string
	smode ifaceMode
}

var (
	selected       bool
	selectLock     sync.Mutex
	multiselection []selection
)

func opsHandler(selPane, auxPane *dirPane, key rune) {
	if !selPane.getLock() {
		return
	}
	defer selPane.setUnlock()

	var opstmp opsMode
	var srctmp []selection

	switch key {
	case 'p', 'm', 'd':
		if multiselection == nil {
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

		srctmp = multiselection

	case 'M', 'R':
		var srcpath string

		switch key {
		case 'M':
			opstmp = opMkdir
			srcpath = filepath.Join(selPane.path, mrinput)

		case 'R':
			opstmp = opRename
			selPane.updateRow(false)

			seltext := selPane.table.GetCell(selPane.row, 0).Text
			srcpath = filepath.Join(selPane.path, seltext)
			checkSelected(selPane.path, seltext, true)

			mrinput = filepath.Join(selPane.path, mrinput)
		}

		srctmp = []selection{{srcpath, selPane.mode}}
	}

	if len(srctmp) == 0 {
		return
	}

	confirmOperation(auxPane, selPane, opstmp, srctmp)
}

func checkSelected(panepath, dirname string, rm bool) bool {
	selectLock.Lock()
	defer selectLock.Unlock()

	if !selected {
		return false
	}

	fullpath := filepath.Join(panepath, dirname)

	for i, spath := range multiselection {
		if spath.path == fullpath {
			if rm {
				multiselection[i] = multiselection[len(multiselection)-1]
				multiselection = multiselection[:len(multiselection)-1]
			}

			return true
		}
	}

	return false
}

func (p *dirPane) multiSelectHandler(all, inverse bool, totalrows int) {
	if !p.getLock() {
		return
	}
	defer p.setUnlock()

	var rows int
	var color tcell.Color

	selected = true

	mselone := !all && !inverse
	mselinv := !all && inverse

	if mselone {
		rows = 1
	} else {
		rows = totalrows
	}

	for i := 0; i < rows; i++ {
		if mselone {
			i, _ = p.table.GetSelection()
		}

		cell := p.table.GetCell(i, 0)
		if cell.Text == "" {
			return
		}

		fullpath := filepath.Join(p.path, cell.Text)
		checksel := checkSelected(p.path, cell.Text, true)

		if checksel && (mselone || mselinv) {
			color = tcell.ColorSkyblue
		} else {
			color = tcell.ColorOrange
			msel := selection{fullpath, p.mode}

			selectLock.Lock()
			multiselection = append(multiselection, msel)
			selectLock.Unlock()
		}

		p.table.SetCell(i, 0, cell.SetTextColor(color))

		if i+1 < totalrows && mselone {
			p.table.Select(i+1, 0)
		}

		if mselone {
			return
		}
	}
}

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
