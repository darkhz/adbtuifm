package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type operation struct {
	id         int
	currFile   int
	totalFile  int
	currBytes  int64
	totalBytes int64
	opmode     opsMode
	transfer   transferMode
	progress   progressMode
	ctx        context.Context
	cancel     context.CancelFunc
}

type ifaceMode int

const (
	mAdb ifaceMode = iota
	mLocal
)

type transferMode int

const (
	adbToAdb transferMode = iota
	adbToLocal
	localToAdb
	localToLocal
)

type opsMode int

const (
	opCopy opsMode = iota
	opMove
	opMkdir
	opRename
	opDelete
)

func (m opsMode) String() string {
	opstr := [...]string{
		"Copy",
		"Move",
		"Mkdir",
		"Rename",
		"Delete",
	}

	return opstr[m]
}

func opString(mode string) string {
	switch mode {
	case "Move", "Delete":
		mode = mode[0 : len(mode)-1]
		fallthrough

	case "Copy":
		mode += "ing"

	default:
		return ""
	}

	return mode
}

var (
	jobNum int

	opPaths    []string
	opPathLock sync.Mutex
)

func newOperation(opmode opsMode) operation {
	transfer := localToLocal
	ctx, cancel := context.WithCancel(context.Background())

	return operation{
		id:         jobNum,
		opmode:     opmode,
		ctx:        ctx,
		cancel:     cancel,
		transfer:   transfer,
		totalBytes: -1,
	}
}

func startOperation(srcPane, dstPane *dirPane, opmode opsMode, overwrite bool, mselect []selection) (string, error) {
	var err error
	var src, dst string

	total := len(mselect)

	op := newOperation(opmode)

	op.opSetStatus(opInProgress, nil)

	for sel, msel := range mselect {
		src = msel.path
		dpath := dstPane.getPath()

		if opmode == opRename {
			dst = mrinput
		} else {
			if dstPane.table == nil {
				dst = dpath
			} else {
				dst = filepath.Join(dpath, filepath.Base(src))
			}
		}

		if isOpen(src, dst, dstPane.table != nil) {
			err = fmt.Errorf("'%s' is open", filepath.Base(src))
			break
		}

		if opmode == opCopy && !overwrite {
			dst, err = altPath(src, dst, dstPane.mode)
			if err != nil {
				break
			}
		}

		if err = isSamePath(src, dst, opmode); err != nil {
			break
		}

		op.transfer = transfermode(opmode, msel.smode, dstPane.mode)

		if err = op.setNewProgress(src, dst, sel, total); err != nil {
			break
		}

		if err = addOpsPath(src, dst); err != nil {
			break
		}

		switch op.transfer {
		case localToLocal:
			err = op.localOps(src, dst)

		default:
			err = op.adbOps(src, dst)
		}

		rmOpsPath(src, dst)

		if err != nil {
			break
		}
	}

	op.opSetStatus(opDone, err)

	reloadpath := trimPath(dst, true)
	if dstPane.getPath() == reloadpath {
		dstPane.ChangeDir(false, false)
	}
	if srcPane.getPath() == reloadpath && srcPane.mode == dstPane.mode {
		srcPane.ChangeDir(false, false)
	}

	return dst, err
}

func transfermode(opmode opsMode, srcMode, dstMode ifaceMode) transferMode {
	switch opmode {
	case opDelete, opRename, opMkdir:
		switch srcMode {
		case mAdb:
			return adbToAdb

		case mLocal:
			return localToLocal
		}

	default:
		switch {
		case srcMode == mLocal && dstMode == mAdb:
			return localToAdb

		case srcMode == mAdb && dstMode == mLocal:
			return adbToLocal

		case srcMode == mAdb && dstMode == mAdb:
			return adbToAdb
		}
	}

	return localToLocal
}

func altPath(src, dst string, iface ifaceMode) (string, error) {
	var try int
	var existerr error

	rel, err := filepath.Rel(src, dst)
	if err != nil {
		return dst, err
	}

	if !strings.HasPrefix(rel, ".") {
		return dst, fmt.Errorf("Cannot Copy %s to %s", src, dst)
	}

	for {
		switch iface {
		case mAdb:
			device, err := getAdb()
			if err != nil {
				return dst, err
			}

			_, existerr = device.Stat(dst)

		case mLocal:
			_, existerr = os.Lstat(dst)
		}

		if existerr != nil {
			break
		}

		if dst[len(dst)-1:] != "_" && try == 0 {
			dst = dst + "_"
			continue
		}

		dst = strings.TrimRightFunc(dst, func(r rune) bool {
			return r < 'A' || r > 'z'
		})

		dst += strconv.Itoa(try)
		try++
	}

	return dst, nil
}

func isOpen(src, dst string, table bool) bool {
	return (checkOpen(src) || checkOpen(dst)) && table
}

func isSamePath(src, dst string, opmode opsMode) error {
	switch opmode {
	case opDelete, opMkdir:
		return nil
	}

	rel, err := filepath.Rel(src, dst)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(rel, "..") {
		return fmt.Errorf("Cannot %s %s to %s", opmode.String(), src, dst)
	}

	return nil
}

func addOpsPath(src, dst string) error {
	opPathLock.Lock()
	defer opPathLock.Unlock()

	for _, path := range opPaths {
		if path == src || path == dst {
			return fmt.Errorf("Already operating on %s", path)
		}
	}

	if dst != "" {
		opPaths = append(opPaths, dst)
	}

	opPaths = append(opPaths, src)

	return nil
}

func rmOpsPath(src, dst string) {
	opPathLock.Lock()
	defer opPathLock.Unlock()

	var paths []string

	for _, path := range opPaths {
		if path == src || path == dst {
			continue
		}

		paths = append(paths, path)
	}

	opPaths = paths
}

func iterOps(all bool, o *operation, cfunc func(row, rows int, op *operation)) {
	app.QueueUpdateDraw(func() {
		rows := opsView.GetRowCount()

		for i := 0; i < rows; i++ {
			cell := opsView.GetCell(i, 0)
			if cell == nil || cell.NotSelectable {
				continue
			}

			ref := cell.GetReference()
			if ref == nil {
				continue
			}

			op := ref.(*operation)
			if o != nil {
				if op != o {
					continue
				}
			}

			cfunc(i, rows, op)

			if !all {
				break
			}
		}
	})
}

func (o *operation) jobFinished() {
	iterOps(false, o, func(row, rows int, op *operation) {
		opsView.RemoveRow(row)
		opsView.RemoveRow(row)
		opsView.RemoveRow(row - 1)

		resetOpsView()

		jobNum = rows - opRowNum
	})
}

func (o *operation) cancelOps() {
	o.cancel()
}

func cancelAllOps() {
	go func() {
		iterOps(true, nil, func(row, rows int, op *operation) {
			op.cancelOps()
		})
	}()
}

func confirmOperation(selPane, auxPane *dirPane, opmode opsMode, overwrite bool, mselect []selection) {
	doFunc := func() {
		if mselect == nil {
			mselect = getselection()
		}

		go startOperation(selPane, auxPane, opmode, overwrite, mselect)
	}

	resetFunc := func() {
		reset(auxPane, selPane)
	}

	if opmode == opRename || opmode == opMkdir {
		doFunc()
		return
	}

	msg := opmode.String() + " selected item(s)"

	if opmode == opCopy && overwrite {
		msg += " (will overwrite existing)"
	}

	msg += " [y/n/S]?"

	showConfirmMsg(msg, doFunc, resetFunc)
}
