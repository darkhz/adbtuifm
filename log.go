package main

import (
	"context"
	"errors"
	"path"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/rivo/tview"
)

var updateOps sync.Mutex

func showError(sterr statusError, msg string, disable bool) {
	if disable {
		return
	}

	switch sterr {
	case openError:
		msg = "Failed to open " + msg
	case statError:
		msg = "Failed to stat " + msg
	case createError:
		msg = "Failed to create " + msg
	case adbError:
		msg = "Failed to connect to ADB"
	case notImplError:
		msg = msg + " is not implemented"
	case unknownError:
		msg = "An unknown error occurred " + msg
	}

	showErrorModal(msg)
}

func (o *opsWork) updateOpsView(col int, msg string) {
	updateOps.Lock()
	defer updateOps.Unlock()

	app.QueueUpdateDraw(func() {
		if col == 0 {
			opsView.SetCell(o.id+1, col, tview.NewTableCell(msg).
				SetAlign(tview.AlignCenter))
			return
		}

		opsView.SetCell(o.id+1, col, tview.NewTableCell(msg).
			SetAlign(tview.AlignCenter).
			SetSelectable(false))
	})
}

func (o *opsWork) opErr(sterr statusError) {
	app.QueueUpdateDraw(func() {
		if sterr == createError {
			_, fname := filepath.Split(path.Clean(o.src))
			showError(sterr, o.dst+fname, false)
		} else if sterr == notImplError {
			showError(sterr, o.ops.String(), false)
		} else {
			showError(sterr, " -- "+o.ops.String()+" on "+o.src, false)
		}
	})
}

func (o *opsWork) opLog(status opStatus, err error) {
	switch status {
	case opInProgress:
		o.updateOpsView(0, strconv.Itoa(o.id))
		o.updateOpsView(1, o.ops.String())

		if o.ops == opDelete {
			o.updateOpsView(2, o.src)
		} else {
			o.updateOpsView(2, o.src+" -> "+o.dst)
		}

		o.updateOpsView(3, "IN PROGRESS")
		opsView.ScrollToEnd()
		jobNum++
	case opDone:
		if errors.Is(err, context.Canceled) {
			o.updateOpsView(3, "[red]CANCELED")
			return
		} else if err != nil {
			o.updateOpsView(3, "[red]ERROR")
			o.opErr(unknownError)
		} else {
			o.updateOpsView(3, "[green]DONE")
		}

		o.cancel()
	}
}
