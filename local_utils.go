package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	adb "github.com/zach-klippenstein/goadb"
)

var pathLock sync.Mutex

func trimPath(testPath string, cdBack bool) string {
	testPath = filepath.Clean(testPath)

	if cdBack {
		testPath = filepath.Dir(testPath)
	}

	if testPath != "/" {
		testPath = testPath + "/"
	}

	return testPath
}

func isLocalSymDir(testPath, name string) bool {
	dpath := fmt.Sprintf("%s%s", testPath, name)

	dpath, err := filepath.EvalSymlinks(dpath)
	if err != nil {
		return false
	}

	entry, err := os.Lstat(dpath)
	if err != nil {
		return false
	}

	if !entry.Mode().IsDir() {
		return false
	}

	return true
}

func (p *dirPane) isDir(testPath string) bool {
	if p.row > len(p.pathList) {
		return false
	}

	name := p.pathList[p.row].Name
	fmode := p.pathList[p.row].Mode

	if fmode&os.ModeSymlink != 0 {
		switch p.mode {
		case mAdb:
			return isAdbSymDir(testPath, name)

		case mLocal:
			return isLocalSymDir(testPath, name)
		}
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

		if entry.IsDir() || isLocalSymDir(testPath, name) {
			dlist = append(dlist, filepath.Join(testPath, name))
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
	var listed bool
	var testPath, prevDir string

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
	p.setPaneSelectable(false)

	switch {
	case cdFwd:
		testPath = trimPath(testPath, false)
		testPath = filepath.Join(testPath, p.table.GetCell(p.row, 0).Text)

	case cdBack:
		prevDir = fmt.Sprintf("%s/", filepath.Base(testPath))
		testPath = trimPath(testPath, cdBack)
	}

	switch p.mode {
	case mAdb:
		_, listed = p.adbListDir(testPath, false)

	case mLocal:
		_, listed = p.localListDir(filepath.FromSlash(testPath), false)
	}

	if !listed {
		return
	}

	p.setPath(filepath.ToSlash(testPath))

	sort.Slice(p.pathList, func(i, j int) bool {
		if p.pathList[i].Mode.IsDir() != p.pathList[j].Mode.IsDir() {
			return p.pathList[i].Mode.IsDir()
		}

		return p.pathList[i].Name < p.pathList[j].Name
	})

	p.createDirList(cdFwd, cdBack, prevDir)
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

func (p *dirPane) createDirList(cdFwd, cdBack bool, prevDir string) {
	app.QueueUpdateDraw(func() {
		var pos int

		p.table.Clear()

		totalrows := len(p.pathList)

		for row, dir := range p.pathList {
			switch {
			case cdBack:
				if dir.Name == prevDir {
					pos = row
				}

			case !cdFwd && !cdBack:
				if p.row >= totalrows {
					pos = totalrows - 1
				} else {
					pos = p.row
				}
			}

			if checkSelected(p.path, dir.Name, false) {
				p.updateDirPane(row, true, nil, dir.Name)
				continue
			}

			p.updateDirPane(row, false, nil, dir.Name)
		}

		if pos == 0 {
			p.table.ScrollToBeginning()
		}

		p.setPaneTitle()
		p.table.Select(pos, 0)
		p.setPaneSelectable(true)
	})
}

func (p *dirPane) setPath(ppath string) {
	pathLock.Lock()
	defer pathLock.Unlock()

	p.path = ppath
}

func (p *dirPane) getPath() string {
	pathLock.Lock()
	defer pathLock.Unlock()

	return p.path
}

func (o *operation) localOps(src, dst string) error {
	var err error

	if o.opmode != opMkdir {
		err = o.getTotalFiles(src)
		if err != nil {
			return err
		}
	}

	switch o.opmode {
	case opMove, opRename:
		err = os.Rename(src, dst)

	case opDelete:
		err = os.RemoveAll(src)

	case opMkdir:
		err = os.Mkdir(src, 0777)

	case opCopy:
		err = o.copyRecursive(src, dst)
	}

	return err
}
