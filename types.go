package main

import (
	"context"

	"github.com/rivo/tview"
	adb "github.com/zach-klippenstein/goadb"
	"golang.org/x/sync/semaphore"
)

type transferMode int

const (
	localToLocal transferMode = iota
	adbToAdb
	localToAdb
	adbToLocal
)

type opsMode int

const (
	opCopy opsMode = iota
	opMove
	opDelete
	opRename
	opNone
)

type ifaceMode int

const (
	mAdb ifaceMode = iota
	mLocal
)

type opStatus int

const (
	opInProgress opStatus = iota
	opDone
)

type dirPane struct {
	row      int
	lock     *semaphore.Weighted
	tbl      *tview.Table
	mode     ifaceMode
	path     string
	apath    string
	dpath    string
	hidden   bool
	selected bool
	pathList []*adb.DirEntry
}

type opsWork struct {
	id        int
	src       string
	dst       string
	ctx       context.Context
	cancel    context.CancelFunc
	ops       opsMode
	transfer  transferMode
	finished  bool
	currFile  int
	totalFile int
}

func (op opsMode) String() string {
	ops := [...]string{"Copy", "Move", "Delete", "Rename"}

	return ops[op]
}
