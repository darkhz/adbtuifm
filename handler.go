package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	adb "github.com/zach-klippenstein/goadb"
)

type selection struct {
	path  string
	smode ifaceMode
}

var (
	selected       bool
	openLock       sync.Mutex
	selectLock     sync.RWMutex
	openFiles      map[string]struct{}
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
		var dir *adb.DirEntry

		if mselone {
			i, _ = p.table.GetSelection()
		}

		cell := p.table.GetCell(i, 0)
		if cell.Text == "" {
			return
		}

		ref := cell.GetReference()
		if ref == nil {
			return
		}

		checksel := false

		dir = ref.(*adb.DirEntry)
		fullpath := filepath.Join(p.path, dir.Name)

		if mselone || mselinv {
			checksel = checkSelected(p.path, dir.Name, true)
		}

		if !checksel {
			addmsel(fullpath, p.mode)
		}

		p.updateDirPane(i, !checksel, dir)

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

func (p *dirPane) openFileHandler() {
	p.updateRef(true)

	if p.entry.Mode.IsDir() {
		return
	}

	name := p.entry.Name
	tpath := filepath.Join("/tmp", name)
	fpath := filepath.Join(p.getPath(), name)

	showInfoMsg(fmt.Sprintf("Transferring '%s', check operations view", name))

	tmpdst, err := startOperation(
		p,
		&dirPane{path: tpath, mode: mLocal},
		opCopy,
		false,
		[]selection{{fpath, p.mode}},
	)
	if err != nil {
		showErrorMsg(
			fmt.Errorf("Unable to open '%s': %s", name, err.Error()),
			false,
		)
		return
	}
	defer os.Remove(tmpdst)

	if checkOpen(fpath) {
		showInfoMsg(fmt.Sprintf("Waiting for process to finish with '%s'", name))
		return
	}
	setOpen(fpath, false)
	defer setOpen(fpath, true)

	showInfoMsg("Opening " + name)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		showErrorMsg(fmt.Errorf("Cannot monitor changes on '%s'", name), false)
		return
	}

	err = watcher.Add(tmpdst)
	if err != nil {
		showErrorMsg(fmt.Errorf("Error monitoring '%s'", name), false)
		return
	}
	defer watcher.Close()

	modify := make(chan bool)
	defer close(modify)

	go func() {
		select {
		case _, ok := <-watcher.Events:
			if !ok {
				return
			}

			modify <- true
		}
	}()

	cmd, err := execCmd(fmt.Sprintf("xdg-open '%s'", tmpdst), "Background", "Local")
	if err != nil {
		showErrorMsg(err, false)
		return
	}

	err = cmd.Wait()
	if err != nil {
		showErrorMsg(err, false)
		return
	}

	select {
	case <-modify:
		_, err = startOperation(
			p,
			&dirPane{path: fpath, mode: p.mode},
			opCopy,
			true,
			[]selection{{tmpdst, mLocal}},
		)

		if err != nil {
			showErrorMsg(
				fmt.Errorf("Unable to save '%s': %s", name, err.Error()),
				false,
			)

			return
		}

		showInfoMsg(fmt.Sprintf("Overwriting modified '%s'", name))

	default:
	}
}

func checkOpen(fpath string) bool {
	openLock.Lock()
	defer openLock.Unlock()

	_, ok := openFiles[fpath]

	return ok
}

func setOpen(fpath string, del bool) {
	openLock.Lock()
	defer openLock.Unlock()

	if del {
		delete(openFiles, fpath)
		return
	}

	openFiles[fpath] = struct{}{}
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
