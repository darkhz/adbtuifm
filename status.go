package main

import (
	"context"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	statuspgs *tview.Pages
	statusmsg *tview.TextView

	mrinput    string
	msgchan    chan string
	entrycache []string

	sctx    context.Context
	scancel context.CancelFunc
)

func startStatus() {
	var cleared bool

	t := time.NewTicker(2 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-sctx.Done():
			return

		case msg := <-msgchan:
			t.Reset(2 * time.Second)

			cleared = false

			app.QueueUpdateDraw(func() {
				statusmsg.SetText(msg)
			})

		case <-t.C:
			if cleared {
				continue
			}

			cleared = true

			app.QueueUpdateDraw(func() {
				statusmsg.Clear()
			})
		}
	}
}

func stopStatus() {
	scancel()
	close(msgchan)
}

func setupStatus() {
	statuspgs = tview.NewPages()

	statusmsg = tview.NewTextView()
	statusmsg.SetDynamicColors(true)

	statuspgs.AddPage("statusmsg", statusmsg, true, true)

	statusmsg.SetBackgroundColor(tcell.Color16)
	statuspgs.SetBackgroundColor(tcell.Color16)

	sctx, scancel = context.WithCancel(context.Background())

	msgchan = make(chan string)
	go startStatus()
}

func getStatusInput(msg string, accept bool) *tview.InputField {
	input := tview.NewInputField()

	input.SetLabel("[::b]" + msg + " ")

	if accept {
		input.SetAcceptanceFunc(tview.InputFieldMaxLength(1))
	}

	input.SetLabelColor(tcell.ColorWhite)
	input.SetBackgroundColor(tcell.Color16)
	input.SetFieldBackgroundColor(tcell.Color16)

	return input
}

func showInfoMsg(msg string) {
	msgchan <- "[::b]" + msg
}

func showErrorMsg(err error, autocomplete bool) {
	if autocomplete {
		return
	}

	msgchan <- "[red::b]" + err.Error()
}

func showConfirmMsg(msg string, alert bool, doFunc, resetFunc func()) {
	input := getStatusInput(msg, true)

	exit := func(reset bool) {
		if reset {
			resetFunc()
		}

		statuspgs.SwitchToPage("statusmsg")
		app.SetFocus(prevPane.table)
	}

	infomsg := func() {
		info := strings.Fields(msg)[0]

		switch info {
		case "Move", "Delete":
			info = info[0 : len(info)-1]
			fallthrough

		case "Copy":
			info += "ing"

		default:
			return
		}

		info += " items"

		msgchan <- "[white]" + info
	}

	confirm := func() {
		var reset bool

		text := input.GetText()
		input.SetText("")

		switch text {
		case "y":
			doFunc()
			infomsg()

			reset = true
			fallthrough

		case "n":
			exit(reset)

		default:
			return
		}
	}

	input.SetChangedFunc(func(text string) {
		if text == "" {
			return
		}

		switch text {
		case "y", "n":
			return

		default:
			input.SetText("")
		}
	})

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			confirm()

		case tcell.KeyEscape, tcell.KeyLeft, tcell.KeyRight:
			exit(false)

		case tcell.KeyUp, tcell.KeyDown:
			exit(false)
			prevPane.table.InputHandler()(event, nil)
		}

		switch event.Rune() {
		case 'S':
			showEditSelections(input)
		}

		return event
	})

	statuspgs.AddAndSwitchToPage("confirm", input, true)
	app.SetFocus(input)
}

func (p *dirPane) showFilterInput() {
	input := getStatusInput("Filter:", false)

	exit := func() {
		statuspgs.SwitchToPage("statusmsg")
		app.SetFocus(prevPane.table)
	}

	input.SetChangedFunc(func(text string) {
		if text == "" {
			p.reselect(false)
			p.table.Select(0, 0)
			p.table.ScrollToBeginning()

			return
		}

		if !p.getLock() {
			return
		}
		defer p.setUnlock()

		p.table.Clear()

		var row int
		for _, dir := range p.pathList {
			if strings.Contains(
				strings.ToLower(dir.Name),
				strings.ToLower(text),
			) {
				sel := checkSelected(p.path, dir.Name, false)

				p.updateDirPane(row, sel, nil, dir)
				row++
			}
		}

		p.table.Select(0, 0)
		p.table.ScrollToBeginning()
	})

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			p.reselect(false)

		case tcell.KeyUp, tcell.KeyDown, tcell.KeyEnter:
			p.table.InputHandler()(event, nil)
			fallthrough

		case tcell.KeyEscape:
			exit()
		}

		return event
	})

	statuspgs.AddAndSwitchToPage("filter", input, true)
	app.SetFocus(input)
}

func showMkdirRenameInput(selPane, auxPane *dirPane, key rune) {
	var title string

	row, _ := selPane.table.GetSelection()
	origname := selPane.table.GetCell(row, 0).Text

	switch key {
	case 'M':
		title = "Make directory:"

	case 'R':
		title = "Rename To:"
	}

	exit := func() {
		statuspgs.SwitchToPage("statusmsg")
		app.SetFocus(selPane.table)
	}

	infomsg := func(newname string) {
		var info string

		switch key {
		case 'M':
			info = "Created " + newname + " in " + selPane.getPath()

		case 'R':
			info = "Renamed " + strings.TrimSuffix(origname, "/") + " to " + newname
		}

		msgchan <- info
	}

	input := getStatusInput(title, false)

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp, tcell.KeyDown:
			prevPane.table.InputHandler()(event, nil)
			exit()

		case tcell.KeyEnter:
			text := input.GetText()
			if text != "" {
				mrinput = text
				opsHandler(selPane, auxPane, key)
			}

			infomsg(text)
			fallthrough

		case tcell.KeyEscape:
			exit()
		}

		return event
	})

	statuspgs.AddAndSwitchToPage("mrinput", input, true)
	app.SetFocus(input)
}

func (p *dirPane) showChangeDirInput() {
	input := getStatusInput("Change Directory to:", false)
	input.SetText(p.path)

	infomsg := func(cdpath string) {
		msgchan <- "Changing directory to " + cdpath
	}

	input.SetAutocompleteFunc(func(current string) (entries []string) {
		var tmpentry []string

		if len(current) == 0 {
			return
		}

		switch p.mode {
		case mAdb:
			tmpentry, _ = p.adbListDir(current, true)
		case mLocal:
			tmpentry, _ = p.localListDir(current, true)
		}

		if tmpentry != nil {
			entrycache = tmpentry
		} else {
			tmpentry = entrycache
		}

		for _, ent := range tmpentry {
			if strings.Index(ent, current) != -1 {
				entries = append(entries, ent)
			}
		}

		return
	})

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			input.Autocomplete()

		case tcell.KeyEnter:
			infomsg(input.GetText())
			p.ChangeDir(false, false, input.GetText())
			fallthrough

		case tcell.KeyEscape:
			statuspgs.SwitchToPage("statusmsg")
			app.SetFocus(p.table)

		case tcell.KeyCtrlW:
			input.SetText(trimPath(input.GetText(), true))
			input.Autocomplete()
			return nil

		case tcell.KeyBackspace, tcell.KeyBackspace2:
			fallthrough

		case tcell.KeyDown, tcell.KeyUp, tcell.KeyLeft, tcell.KeyRight:
			return event
		}

		switch event.Rune() {
		default:
			input.Autocomplete()
		}

		return event
	})

	statuspgs.AddAndSwitchToPage("cdinput", input, true)
	app.SetFocus(input)
}