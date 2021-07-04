package main

import (
	"context"
	"errors"
	"strconv"

	"github.com/rivo/tview"
)

func showError(err error, msg string) {
	if err != nil {
		if msg != "" {
			msg = err.Error() + " -- " + msg
		} else {
			msg = err.Error()
		}
	}

	showErrorModal(msg)
}

func (o *opsWork) updateOpsView(col int, msg string) {
	if col == 0 {
		opsView.SetCell(o.id+1, col, tview.NewTableCell(msg).
			SetAlign(tview.AlignCenter))
		return
	}

	opsView.SetCell(o.id+1, col, tview.NewTableCell(msg).
		SetAlign(tview.AlignCenter).
		SetSelectable(false))
}

func (o *opsWork) opLog(status opStatus, err error) {
	app.QueueUpdateDraw(func() {
		switch status {
		case openError:
			showError(err, "Failed to open")
		case statError:
			showError(err, "Failed to stat")
		case createError:
			showError(err, "Failed to create")
		case adbError:
			showError(err, "Failed to connect to ADB")
		case notImplError:
			showError(nil, o.ops.String()+" is not implemented")
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
				showError(err, "Jobs failed")
				o.updateOpsView(3, "[red]ERROR")
				return
			}

			o.updateOpsView(3, "[green]DONE")
		}
	})
}
