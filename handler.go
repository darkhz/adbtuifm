package main

import (
	"path/filepath"
	"sync"

	"github.com/rivo/tview"
)

type selection struct {
	path  string
	smode ifaceMode
}

var (
	selected       bool
	selectLock     sync.RWMutex
	multiselection map[string]ifaceMode
)

func opsHandler(selPane, auxPane *dirPane, key rune) {
	if !selPane.getLock() {
		return
	}
	defer selPane.setUnlock()

	var opstmp opsMode
	var overwrite bool
	var srctmp []selection

	switch key {
	case 'p', 'P', 'm', 'd':
		if len(multiselection) == 0 {
			return
		}

		switch key {
		case 'P':
			overwrite = true
			fallthrough

		case 'p':
			opstmp = opCopy

		case 'm':
			opstmp = opMove

		case 'd':
			opstmp = opDelete
		}

		srctmp = nil

	case 'M', 'R':
		var srcpath string

		switch key {
		case 'M':
			opstmp = opMkdir
			srcpath = filepath.Join(selPane.path, mrinput)

		case 'R':
			opstmp = opRename
			selPane.updateRef(false)

			srcpath = filepath.Join(selPane.path, selPane.entry.Name)
			checkSelected(selPane.path, selPane.entry.Name, true)

			mrinput = filepath.Join(selPane.path, mrinput)
		}

		srctmp = []selection{{srcpath, selPane.mode}}
	}

	confirmOperation(auxPane, selPane, opstmp, overwrite, srctmp)
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

func (p *dirPane) multiSelectHandler(all, inverse bool, totalrows int) {
	if !p.getLock() {
		return
	}
	defer p.setUnlock()

	pos, _ := p.table.GetSelection()

	selected = true

	mselone := !all && !inverse
	mselinv := !all && inverse

	for i := 0; i < totalrows; i++ {
		if mselone {
			i, _ = p.table.GetSelection()
		}

		cell := p.table.GetCell(i, 0)
		if cell.Text == "" {
			return
		}

		checksel := false
		fullpath := filepath.Join(p.path, cell.Text)

		if mselone || mselinv {
			checksel = checkSelected(p.path, cell.Text, true)
		}

		if !checksel {
			addmsel(fullpath, p.mode)
		}

		cells := make([]*tview.TableCell, infocols)

		cells[0] = cell
		for col := 1; col < infocols; col++ {
			cells[col] = p.table.GetCell(i, col)
		}

		p.updateDirPane(i, !checksel, cells, nil)

		if mselone {
			if i+1 < totalrows {
				p.table.Select(i+1, 0)
				return
			}

			break
		}
	}

	p.table.Select(pos, 0)
}

func checkSelected(panepath, dirname string, rm bool) bool {
	if !selected {
		return false
	}

	fullpath := filepath.Join(panepath, dirname)

	ok := checkmsel(fullpath)

	if ok && rm {
		delmsel(fullpath)
	}

	return ok
}

func delmsel(fullpath string) {
	selectLock.Lock()
	defer selectLock.Unlock()

	delete(multiselection, fullpath)
}

func addmsel(fullpath string, mode ifaceMode) {
	selectLock.Lock()
	defer selectLock.Unlock()

	multiselection[fullpath] = mode
}

func checkmsel(fullpath string) bool {
	selectLock.RLock()
	defer selectLock.RUnlock()

	_, ok := multiselection[fullpath]

	return ok
}

func getselection() []selection {
	selectLock.RLock()
	defer selectLock.RUnlock()

	var s []selection

	for path, smode := range multiselection {
		s = append(s, selection{path, smode})
	}

	return s
}
