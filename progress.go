package main

import (
	"fmt"
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
