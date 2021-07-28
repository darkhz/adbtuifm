package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/rivo/tview"
	"golang.org/x/sync/semaphore"
)

var updateLock = semaphore.NewWeighted(1)

func showError(err error, autocomplete bool) {
	if autocomplete {
		return
	}

	app.QueueUpdateDraw(func() {
		showErrorModal(err.Error())
	})
}

func (o *opsWork) updateOpsView(col int, msg string) {
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

func (o *opsWork) opLog(status opStatus, err error) {
	for {
		if getUpdateLock() {
			break
		}
	}
	defer setUpdateUnlock()

	switch status {
	case opInProgress:
		path := o.src
		jobNum++

		if o.ops != opDelete {
			path = fmt.Sprintf("%s -> %s", o.src, o.dst)
		}

		o.updateOpsView(0, strconv.Itoa(o.id))
		o.updateOpsView(1, o.ops.String())
		o.updateOpsView(2, path)
		o.updateOpsView(3, "IN PROGRESS")
	case opDone:
		o.cancel()

		switch err {
		case nil:
			o.updateOpsView(3, "[green]DONE")
		case context.Canceled:
			o.updateOpsView(3, "[red]CANCELED")
		default:
			o.updateOpsView(3, "[red]ERROR")
			showError(err, false)
		}

		jobFinished(o.id)
	}
}

func getUpdateLock() bool {
	return updateLock.TryAcquire(1)
}

func setUpdateUnlock() {
	updateLock.Release(1)
}
