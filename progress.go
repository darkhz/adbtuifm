package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/machinebox/progress"
)

var (
	resume     bool
	resumeLock sync.Mutex
)

func (o *operation) startProgress(curfile, totalfile, selindex int, size int64, pcnt progress.Counter, recursive bool) {
	go func() {
		var prog string

		ctx, cancel := context.WithCancel(context.Background())
		pchan := progress.NewTicker(ctx, pcnt, size, 2*time.Second)

		for p := range pchan {
			if curfile == totalfile || o.checkSelIndex(selindex) {
				cancel()
				return
			}

			select {
			case <-o.ctx.Done():
				cancel()
				return

			default:
			}

			if !getResume() {
				continue
			}

			if !recursive {
				prog = fmt.Sprintf("%d%%", int(p.Percent()))
			} else {
				prog = fmt.Sprintf("File %d (%d%%) of %d", curfile+1, int(p.Percent()), totalfile)
			}

			o.updateOpsView(3, prog)
		}
	}()

	o.currFile++
}

func (o *operation) checkSelIndex(selindex int) bool {
	o.selLock.Lock()
	defer o.selLock.Unlock()

	if o.selindex > selindex {
		return true
	}

	return false
}

func (o *operation) setSelIndex(selindex int) {
	o.selLock.Lock()
	defer o.selLock.Unlock()

	o.selindex = selindex
}

func (o *operation) setNewProgress(src, dst string, selindex, seltotal int) (string, string) {
	var tpath string

	dst = filepath.Join(dst, filepath.Base(src))

	switch o.opmode {
	case opDelete, opMkdir:
		tpath = src

	case opRename:
		dst = mrinput
		fallthrough

	default:
		tpath = fmt.Sprintf("%s -> %s", src, dst)
	}

	o.currFile = 0
	o.totalFile = 0
	o.setSelIndex(selindex)

	if seltotal > 1 {
		tpath = fmt.Sprintf("%s (%d of %d)", tpath, selindex+1, seltotal)
	}

	o.updateOpsView(2, tpath)
	o.updateOpsView(3, "Calculating files..")

	return src, dst
}

func setResume(status bool) {
	resumeLock.Lock()
	defer resumeLock.Unlock()

	resume = status
}

func getResume() bool {
	resumeLock.Lock()
	defer resumeLock.Unlock()

	return resume
}
