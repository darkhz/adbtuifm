package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/rivo/tview"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/semaphore"
)

type progressMode struct {
	text *tview.TableCell
	prog *tview.TableCell
	pbar *progressbar.ProgressBar
	lock *semaphore.Weighted
}

type opStatus int

const (
	opInProgress opStatus = iota
	opDone
)

const opRowNum = 3

var (
	progWidth  int
	updateLock sync.Mutex
)

func (o operation) getDescription() string {
	if o.totalFile <= 1 {
		return ""
	}

	return fmt.Sprintf("[%d/%d]", o.currFile+1, o.totalFile)
}

func (o *operation) createPb() {
	if o.progress.lock == nil {
		o.progress.lock = semaphore.NewWeighted(1)
	}

	if !o.progress.lock.TryAcquire(1) {
		return
	}
	defer o.progress.lock.Release(1)

	o.progress.pbar = progressbar.NewOptions64(
		o.totalBytes,
		progressbar.OptionFullWidth(),
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
		if o.totalBytes > -1 {
			o.progress.prog.SetText(string(b))
		} else {
			o.progress.prog.SetText("  " + string(b))
		}
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

	tpath = "  " + opString(opstr) + " "

	switch o.opmode {
	case opDelete, opMkdir:
		tpath += srcstr

	case opCopy:
		pstr = "Calculating.."
		fallthrough

	default:
		tpath += "'" + srcstr + "' to '" + dstdir + "'"
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
				if !o.progress.lock.TryAcquire(1) {
					return
				}
				defer o.progress.lock.Release(1)

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
		jobNum += opRowNum
		o.updateOpsView(true)

	case opDone:
		o.cancel()

		if err != nil {
			if err != context.Canceled {
				e := errors.New("Job #" + strconv.Itoa((o.id+1)/opRowNum) + ": " + err.Error())

				showErrorMsg(e, false)
			}
		}

		o.jobFinished()
	}

	if o.opmode != opRename && o.opmode != opMkdir {
		if jobNum > 0 {
			text := strconv.Itoa(jobNum/opRowNum) + " job(s) are running"
			msgchan <- message{text, true}
		} else {
			msgchan <- message{"", true}
		}
	}
}

func (o *operation) updateOpsView(init bool, msg ...string) {
	app.QueueUpdateDraw(func() {
		if init {
			opsView.SetCell(o.id, 0, tview.NewTableCell("").
				SetSelectable(false))

			opsView.SetCell(o.id, 1, tview.NewTableCell("").
				SetSelectable(false))

			return
		}

		o.progress.prog = tview.NewTableCell("")
		o.progress.text = tview.NewTableCell("")

		opsView.SetCell(o.id+1, 0, tview.NewTableCell("*").
			SetReference(o).
			SetSelectable(true))

		opsView.SetCell(o.id+1, 1, o.progress.text.
			SetText(msg[0]).
			SetExpansion(1).
			SetReference(msg[0]).
			SetSelectable(false).
			SetAlign(tview.AlignLeft))

		opsView.SetCell(o.id+2, 0, tview.NewTableCell("").
			SetSelectable(false))

		opsView.SetCell(o.id+2, 1, o.progress.prog.
			SetText(msg[1]).
			SetExpansion(1).
			SetSelectable(false).
			SetAlign(tview.AlignLeft))
	})
}
