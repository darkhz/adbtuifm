package main

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dolmen-go/contextio"
	adb "github.com/zach-klippenstein/goadb"
)

var setHidden bool

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

func checkPath(testPath string) bool {
	fi, err := os.Stat(testPath)
	if err != nil {
		return false
	}
	if !fi.Mode().IsDir() {
		return false
	}

	return true
}

func (p *dirPane) ChangeDir(cdFwd bool, cdBack bool) {
	row := p.row
	testPath := p.path

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
		p.adbListDir(testPath)
	case mLocal:
		p.localListDir(filepath.FromSlash(testPath))
	}

	list := p.pathList

	p.path = filepath.ToSlash(trimPath(testPath, false))

	if list == nil {
		return
	}

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
	_, fname := filepath.Split(o.src)

	o.opLog(opInProgress, nil)

	switch o.ops {
	case opMove:
		err := os.Rename(o.src, o.dst+fname)
		if err != nil {
			o.opErr(unknownError)
		}

		o.opLog(opDone, err)
		return
	case opDelete:
		err := os.Remove(o.src)
		if err != nil {
			o.opErr(unknownError)
		}

		o.opLog(opDone, err)
		return
	}

	srcFile, err := os.Open(o.src)
	if err != nil {
		o.opErr(openError)
		return
	}
	defer srcFile.Close()

	dstFile, err := os.Create(o.dst + fname)
	if err != nil {
		o.opErr(createError)
		return
	}
	defer dstFile.Close()

	cioIn := contextio.NewReader(o.ctx, srcFile)
	_, err = io.Copy(dstFile, cioIn)

	o.opLog(opDone, err)
}
