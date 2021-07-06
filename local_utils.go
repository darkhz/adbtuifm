package main

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dolmen-go/contextio"
	"github.com/zach-klippenstein/goadb"
)

var setHidden bool = true

func isDir(pane *dirPane, testPath string) bool {
	name := pane.pathList[pane.row].Name
	fmode := pane.pathList[pane.row].Mode

	if pane.mode == mAdb && fmode&os.ModeSymlink != 0 {
		return isSymDir(testPath, name)
	}

	if !fmode.IsDir() {
		return false
	}

	return true
}

func (p *dirPane) localListDir(testPath string) {
	fi, err := os.Lstat(testPath)
	if err != nil {
		return
	}

	file, err := os.Open(testPath)
	if err != nil {
		return
	}
	defer file.Close()

	list, _ := file.Readdirnames(0)

	for _, name := range list {
		var d adb.DirEntry

		if setHidden && strings.HasPrefix(name, ".") {
			continue
		}

		fi, err = os.Stat(testPath + name)
		if err != nil {
			return
		}

		mode := fi.Mode()
		if mode.IsDir() {
			name = name + "/"
		}

		d.Name = name
		d.Mode = mode
		d.Size = int32(fi.Size())
		d.ModifiedAt = fi.ModTime()

		p.pathList = append(p.pathList, &d)
	}
}

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

func (p *dirPane) ChangeDir(cdFwd bool, cdBack bool) {
	row := p.row
	testPath := p.path

	if cdFwd && p.pathList != nil && !isDir(p, testPath) {
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
		p.adbListDir(testPath)
	case mLocal:
		p.localListDir(filepath.FromSlash(testPath))
	}

	list := p.pathList

	p.path = filepath.ToSlash(trimPath(testPath, false))

	sort.Slice(list, func(i, j int) bool {
		if list[i].Mode.IsDir() != list[j].Mode.IsDir() {
			return list[i].Mode.IsDir()
		}

		return list[i].Name < list[j].Name
	})

	p.tbl.Clear()

	row = 0
	for _, d := range p.pathList {
		p.updateDirPane(row, d.Name)
		row++
	}

	setPaneTitle(p)
	p.tbl.Select(0, 0)
	p.tbl.ScrollToBeginning()
}

func (o *opsWork) localOps() {
	if o.ops == opMove || o.ops == opDelete {
		o.opLog(notImplError, nil)
		return
	}

	srcFile, err := os.Open(o.src)
	if err != nil {
		o.opLog(openError, err)
		return
	}
	defer srcFile.Close()

	_, fname := filepath.Split(o.src)

	dstFile, err := os.Create(o.dst + fname)
	if err != nil {
		o.opLog(createError, err)
		return
	}
	defer dstFile.Close()

	o.opLog(opInProgress, nil)

	cioIn := contextio.NewReader(o.ctx, srcFile)
	_, err = io.Copy(dstFile, cioIn)

	o.opLog(opDone, err)
}
