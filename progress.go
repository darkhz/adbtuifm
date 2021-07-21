package main

import (
	"fmt"
	"time"

	"github.com/machinebox/progress"
)

func (o *opsWork) startProgress(curNum int, size int64, pcnt progress.Counter, recursive bool) {
	go func() {
		var prog string

		pchan := progress.NewTicker(o.ctx, pcnt, size, 2*time.Second)
		for p := range pchan {
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
