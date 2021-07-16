package main

import (
	"strconv"
	"time"

	"github.com/machinebox/progress"
)

func (o *opsWork) startProgress(curNum int, size int64, pcnt progress.Counter, recursive bool) {
	go func() {
		var prog string

		pchan := progress.NewTicker(o.ctx, pcnt, size, 2*time.Second)
		for p := range pchan {
			if recursive {
				prog = "File " + strconv.Itoa(curNum+1) + " (" + strconv.Itoa(int(p.Percent())) + "%) of " + strconv.Itoa(o.totalFile)
			} else {
				prog = strconv.Itoa(int(p.Percent())) + "%"
			}

			o.updateOpsView(3, prog)
		}
	}()

	o.currFile++
}
