package main

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	adb "github.com/zach-klippenstein/goadb"
)

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

	fi, err := os.Lstat(testPath)
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

	list, _ := file.Readdirnames(0)

	for _, name := range list {
		var d adb.DirEntry

		if p.hidden && strings.HasPrefix(name, ".") {
			continue
		}

		fi, err = os.Lstat(testPath + name)
		if err != nil {
			showError(err, autocomplete)
			return nil, false
		}

		mode := fi.Mode()
		if mode.IsDir() {
			dlist = append(dlist, testPath+name)
			name = name + "/"
		}

		if autocomplete {
			continue
		}

		d.Name = name
		d.Mode = mode
		d.Size = int32(fi.Size())
		d.ModifiedAt = fi.ModTime()

		p.pathList = append(p.pathList, &d)
	}

	return dlist, true
}

func (p *dirPane) ChangeDir(cdFwd bool, cdBack bool) {
	row := p.row
	origPath := p.path
	testPath := trimPath(p.path, false)

	if cdFwd && p.pathList != nil && !p.isDir(testPath) {
		return
	}

	p.pathList = nil

	if cdFwd && row != -1 {
		testPath = testPath + p.tbl.GetCell(row, 0).Text
	} else if cdBack {
		testPath = trimPath(testPath, cdBack)
	}

	switch p.mode {
	case mAdb:
		origPath = p.apath
		_, cdFwd = p.adbListDir(testPath, false)
	case mLocal:
		origPath = p.dpath
		_, cdFwd = p.localListDir(filepath.FromSlash(testPath), false)
	}

	if p.pathList == nil && !cdFwd {
		p.path = filepath.ToSlash(trimPath(origPath, false))
		return
	}

	p.path = filepath.ToSlash(trimPath(testPath, false))

	switch p.mode {
	case mAdb:
		p.apath = p.path
	case mLocal:
		p.dpath = p.path
	}

	sort.Slice(p.pathList, func(i, j int) bool {
		if p.pathList[i].Mode.IsDir() != p.pathList[j].Mode.IsDir() {
			return p.pathList[i].Mode.IsDir()
		}

		return p.pathList[i].Name < p.pathList[j].Name
	})

	p.tbl.Clear()

	for row, dir := range p.pathList {
		p.updateDirPane(row, dir.Name)
	}

	p.setPaneTitle()
	p.tbl.Select(0, 0)
	p.tbl.ScrollToBeginning()
}
