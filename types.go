package main

import (
	"context"

	"github.com/rivo/tview"
	"github.com/zach-klippenstein/goadb"
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
	opNone
)

type ifaceMode int

const (
	mAdb ifaceMode = iota
	mLocal
)

type opStatus int

const (
	openError opStatus = iota
	statError
	createError
	unknownError
	notImplError
	adbError

	opInProgress
	opDone
)

type dirPane struct {
	tbl      *tview.Table
	row      int
	mode     ifaceMode
	path     string
	apath    string
	dpath    string
	pathList []*adb.DirEntry
}

type opsWork struct {
	id       int
	src      string
	dst      string
	pane     *dirPane
	ctx      context.Context
	cancel   context.CancelFunc
	ops      opsMode
	transfer transferMode
	finished bool
}

func (op opsMode) String() string {
	ops := [...]string{"Copy", "Move", "Delete"}

	return ops[op]
}
