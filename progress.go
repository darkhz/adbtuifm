package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/rivo/tview"
	"github.com/schollz/progressbar/v3"
)

type progressMode struct {
	text *tview.TableCell
	prog *tview.TableCell
	pbar *progressbar.ProgressBar
}

type opStatus int

const (
	opInProgress opStatus = iota
	opDone
)

var updateLock sync.Mutex

func (o operation) getDescription() string {
	if o.totalBytes < 0 {
		return "-- In progress --"
	}

	if o.totalFile <= 1 {
		return ""
	}

	return fmt.Sprintf("[%d/%d]", o.currFile+1, o.totalFile)
}

func (o *operation) createPb() {
	o.progress.pbar = progressbar.NewOptions64(
		o.totalBytes,
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetWriter(o),
		progressbar.OptionSpinnerType(34),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionThrottle(200*time.Millisecond),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetDescription(o.getDescription()),
	)
}

func (o *operation) Write(b []byte) (n int, err error) {
	select {
	case <-o.ctx.Done():
		return

	default:
	}

	app.QueueUpdateDraw(func() {
		o.progress.prog.SetText(string(b))
	})

	return 0, nil
}

func (o *operation) updatePb() {
	o.currFile++
	o.progress.pbar.Describe(o.getDescription())
}

func (o *operation) setNewProgress(src, dst string, selindex, seltotal int) error {
	var pstr string
	var tpath string

	opstr := o.opmode.String()
	srcstr := tview.Escape(filepath.Base(src))
	dstdir := tview.Escape(trimPath(filepath.Dir(dst), false))

	tpath = opString(opstr) + " "

	switch o.opmode {
	case opDelete, opMkdir:
		tpath += srcstr

	case opCopy:
		pstr = "Calculating.."
		fallthrough

	default:
		tpath += srcstr + " to " + dstdir
	}

	o.currFile = 0
	o.totalFile = 0

	if seltotal > 1 {
		tpath += fmt.Sprintf(" (%d of %d)", selindex+1, seltotal)
	}

	o.updateOpsView(false, tpath, pstr)

	if o.opmode != opRename || o.opmode != opMkdir {
		if o.opmode == opCopy {
			err := o.getTotalFiles(src)
			if err != nil {
				return err
			}
		}

		o.createPb()

		if o.opmode != opCopy || o.transfer == adbToAdb {
			go func() {
				for {
					select {
					case <-o.ctx.Done():
						return

					default:
					}

					o.progress.pbar.Add64(1)
					time.Sleep(20 * time.Millisecond)
				}
			}()
		}
	}

	return nil
}

func (o *operation) opSetStatus(status opStatus, err error) {
	updateLock.Lock()
	defer updateLock.Unlock()

	switch status {
	case opInProgress:
		jobNum += 2
		o.updateOpsView(true)

	case opDone:
		o.cancel()

		if err != nil {
			if err != context.Canceled {
				showErrorMsg(err, false)
			}
		}

		o.jobFinished()
	}
}

func (o *operation) updateOpsView(init bool, msg ...string) {
	app.QueueUpdateDraw(func() {
		if init {
			opsView.SetCell(o.id, 0, tview.NewTableCell("").
				SetSelectable(false))
			opsView.SetCell(o.id, 1, tview.NewTableCell("").
				SetSelectable(false))

			opsView.SetCell(o.id+1, 0, tview.NewTableCell("*").
				SetReference(o))

			return
		}

		o.progress.prog = tview.NewTableCell("")
		o.progress.text = tview.NewTableCell("")

		opsView.SetCell(o.id+1, 1, o.progress.text.
			SetText(msg[0]).
			SetExpansion(1).
			SetSelectable(false).
			SetAlign(tview.AlignLeft))

		opsView.SetCell(o.id+1, 2, o.progress.prog.
			SetText(msg[1]).
			SetExpansion(1).
			SetSelectable(false).
			SetAlign(tview.AlignRight))
	})
}
