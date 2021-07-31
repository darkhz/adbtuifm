package main

import (
	"fmt"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/machinebox/progress"
)

var resume bool
var resumeLock sync.Mutex

func (o *opsWork) startProgress(curNum int, size int64, pcnt progress.Counter, recursive bool) {
	go func() {
		var prog string

		pchan := progress.NewTicker(o.ctx, pcnt, size, 2*time.Second)
		for p := range pchan {
			if !getProgress() {
				continue
			}

			if recursive {
				prog = fmt.Sprintf("File %d (%d%%) of %d", curNum+1, int(p.Percent()), o.totalFile)
			} else {
				prog = fmt.Sprintf("%d%%", int(p.Percent()))
			}

			o.updateOpsView(3, prog)
		}
	}()

	o.currFile++
}

func (o *opsWork) updatePathProgress(src, dst string, count, total int) string {
	tpath := src
	dst = filepath.Join(dst, path.Base(src))

	if o.ops != opDelete {
		tpath = fmt.Sprintf("%s -> %s", src, dst)
	}

	o.src = src
	o.dst = dst

	o.currFile = 0
	o.totalFile = 0

	if total > 1 {
		tpath = fmt.Sprintf("%s (%d of %d)", tpath, count+1, total)
	}

	o.updateOpsView(2, tpath)

	return dst
}

func setProgress(status bool) {
	resumeLock.Lock()
	defer resumeLock.Unlock()

	resume = status
}

func getProgress() bool {
	resumeLock.Lock()
	defer resumeLock.Unlock()

	return resume
}
