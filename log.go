package main

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"github.com/rivo/tview"
)

var updateOps sync.Mutex

func showError(err error, autocomplete bool) {
	if autocomplete {
		return
	}

	showErrorModal(err.Error())
}

func (o *opsWork) opErr(err error) {
	app.QueueUpdateDraw(func() {
		showError(err, false)
	})
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
		if err != nil {
			if errors.Is(err, context.Canceled) {
				o.updateOpsView(3, "[red]CANCELED")
				return
			}

			o.updateOpsView(3, "[red]ERROR")
			o.opErr(err)
		} else {
			o.updateOpsView(3, "[green]DONE")
		}

		o.cancel()
	}
}
