package main

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	adb "github.com/zach-klippenstein/goadb"
)

var hiddenlock sync.Mutex

func trimPath(testPath string, cdBack bool) string {
	testPath = path.Clean(testPath)

	if cdBack {
		testPath = path.Dir(testPath)
	}

	if testPath != "/" {
		testPath = testPath + "/"
	}

	return testPath
}

func (o *opsWork) localOps() {
	var err error

	o.opLog(opInProgress, nil)

	switch o.ops {
	case opMove:
		err = os.Rename(o.src, o.dst)
	case opDelete:
		err = os.RemoveAll(o.src)
	case opCopy:
		err = o.copyRecursive(o.src, o.dst)
	}

	o.opLog(opDone, err)
}

func (p *dirPane) isDir(testPath string) bool {
	name := p.pathList[p.row].Name
	fmode := p.pathList[p.row].Mode

	if p.mode == mAdb && fmode&os.ModeSymlink != 0 {
		return isSymDir(testPath, name)
	}

	if !fmode.IsDir() {
		return false
	}

	return true
}

func (p *dirPane) localListDir(testPath string, autocomplete bool) ([]string, bool) {
	var dlist []string

	_, err := os.Lstat(testPath)
	if err != nil {
		showError(err, autocomplete)
		return nil, false
	}

	file, err := os.Open(testPath)
	if err != nil {
		showError(err, autocomplete)
		return nil, false
	}
	defer file.Close()

	list, _ := ioutil.ReadDir(testPath)

	for _, entry := range list {
		var d adb.DirEntry

		name := entry.Name()

		if p.getHidden() && strings.HasPrefix(name, ".") {
			continue
		}

		if entry.IsDir() {
			dlist = append(dlist, testPath+name)
			name = name + "/"
		}

		if autocomplete {
			continue
		}

		d.Name = name
		d.Mode = entry.Mode()
		d.Size = int32(entry.Size())
		d.ModifiedAt = entry.ModTime()

		p.pathList = append(p.pathList, &d)
	}

	return dlist, true
}

func (p *dirPane) doChangeDir(cdFwd bool, cdBack bool, tpath ...string) {
	var testPath string

	p.updateRow(true)

	if tpath != nil {
		testPath = tpath[0]
	} else {
		testPath = p.path
	}

	if cdFwd && p.pathList != nil && !p.isDir(testPath) {
		return
	}

	p.pathList = nil

	if cdFwd {
		testPath = trimPath(testPath, false)
		testPath = path.Join(testPath, p.tbl.GetCell(p.row, 0).Text)
	} else if cdBack {
		testPath = trimPath(testPath, cdBack)
	}

	p.setPaneListStatus(true)

	switch p.mode {
	case mAdb:
		_, cdFwd = p.adbListDir(testPath, false)
	case mLocal:
		_, cdFwd = p.localListDir(filepath.FromSlash(testPath), false)
	}

	if !cdFwd {
		return
	}

	p.path = filepath.ToSlash(testPath)

	sort.Slice(p.pathList, func(i, j int) bool {
		if p.pathList[i].Mode.IsDir() != p.pathList[j].Mode.IsDir() {
			return p.pathList[i].Mode.IsDir()
		}

		return p.pathList[i].Name < p.pathList[j].Name
	})

	app.QueueUpdateDraw(func() {
		p.tbl.Clear()

		for row, dir := range p.pathList {
			p.updateDirPane(row, dir.Name)
		}

		p.setPaneTitle()
		p.tbl.Select(0, 0)
		p.tbl.ScrollToBeginning()

		p.setPaneListStatus(false)
	})
}

func (p *dirPane) ChangeDir(cdFwd bool, cdBack bool, tpath ...string) {
	go func() {
		if !p.getLock() {
			return
		}
		defer p.setUnlock()

		p.doChangeDir(cdFwd, cdBack, tpath...)
	}()
}

func (p *dirPane) getLock() bool {
	return p.lock.TryAcquire(1)
}

func (p *dirPane) setUnlock() {
	p.lock.Release(1)
}
