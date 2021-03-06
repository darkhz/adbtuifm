package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/darkhz/tview"
	adb "github.com/zach-klippenstein/goadb"
)

type message struct {
	text    string
	persist bool
}

var (
	statuspgs *tview.Pages
	statusmsg *tview.TextView

	mrinput    string
	msgchan    chan message
	entrycache []string

	sctx    context.Context
	scancel context.CancelFunc
)

func startStatus() {
	var text string
	var cleared bool

	t := time.NewTicker(2 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-sctx.Done():
			return

		case msg, ok := <-msgchan:
			if !ok {
				return
			}

			t.Reset(2 * time.Second)

			cleared = false

			if msg.persist {
				text = msg.text
			}

			app.QueueUpdateDraw(func() {
				if !msg.persist || (msg.text == "" && msg.persist) {
					statusmsg.SetText(msg.text)
				}
			})

		case <-t.C:
			if cleared {
				continue
			}

			cleared = true

			app.QueueUpdateDraw(func() {
				statusmsg.SetText(text)
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

	statusmsg.SetBackgroundColor(tcell.ColorDefault)
	statuspgs.SetBackgroundColor(tcell.ColorDefault)

	sctx, scancel = context.WithCancel(context.Background())

	msgchan = make(chan message)
	go startStatus()
}

func getStatusInput(msg string, accept bool) *tview.InputField {
	input := tview.NewInputField()

	input.SetLabel("[::b]" + msg + " ")

	if accept {
		input.SetAcceptanceFunc(tview.InputFieldMaxLength(1))
	}

	input.SetLabelColor(tcell.ColorWhite)
	input.SetBackgroundColor(tcell.ColorDefault)
	input.SetFieldBackgroundColor(tcell.ColorDefault)

	return input
}

func showInfoMsg(msg string) {
	msgchan <- message{"[::b]" + tview.Escape(msg), false}
}

func showErrorMsg(err error, autocomplete bool) {
	if autocomplete {
		return
	}

	msgchan <- message{"[red::b]" + tview.Escape(err.Error()), false}
}

func showConfirmMsg(msg string, doFunc, resetFunc func()) {
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

		info = opString(info)

		if info == "" {
			return
		}

		info += " items"

		msgchan <- message{"[white]" + info, false}
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
	var regex bool

	input := getStatusInput("", false)

	inputlabel := func() {
		var mode string

		if regex {
			mode = "regex"
		} else {
			mode = "normal"
		}

		label := fmt.Sprintf("[::b]Filter (%s): ", mode)
		input.SetLabel(label)
	}

	exit := func() {
		p.finput = input.GetText()
		statuspgs.SwitchToPage("statusmsg")
		app.SetFocus(prevPane.table)
	}

	filter := func(row int, dir *adb.DirEntry) {
		sel := checkSelected(p.path, dir.Name, false)
		p.updateDirPane(row, sel, dir)
	}

	input.SetChangedFunc(func(text string) {
		if text == "" {
			p.reselect(true)
			p.table.Select(0, 0)
			p.table.ScrollToBeginning()

			return
		}

		if !p.getLock() {
			return
		}
		defer p.setUnlock()

		p.filter = true

		p.table.Clear()

		var row int
		for _, dir := range p.pathList {
			if regex {
				re, err := regexp.Compile(text)
				if err != nil {
					return
				}

				match := re.Match([]byte(dir.Name))
				if match {
					filter(row, dir)
					row++
				}

				continue
			}

			if strings.Contains(
				strings.ToLower(dir.Name),
				strings.ToLower(text),
			) {
				filter(row, dir)
				row++
			}
		}

		p.table.Select(0, 0)
		p.table.ScrollToBeginning()
	})

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			p.reselect(true)

		case tcell.KeyCtrlF:
			regex = !regex
			inputlabel()

		case tcell.KeyUp, tcell.KeyDown:
			p.table.InputHandler()(event, nil)
			fallthrough

		case tcell.KeyEnter, tcell.KeyEscape:
			exit()
		}

		return event
	})

	input.SetText(p.finput)

	inputlabel()

	statuspgs.AddAndSwitchToPage("filter", input, true)
	app.SetFocus(input)
}

func showMkdirRenameInput(selPane, auxPane *dirPane, key rune) {
	var title string
	var rename bool

	row, _ := selPane.table.GetSelection()

	ref := selPane.table.GetCell(row, 0).GetReference()
	if ref == nil {
		return
	}

	origname := ref.(*adb.DirEntry).Name

	switch key {
	case 'M':
		rename = false
		title = "Make directory:"

	case 'R':
		rename = true
		title = "Rename To:"
	}

	exit := func() {
		statuspgs.SwitchToPage("statusmsg")
		app.SetFocus(selPane.table)
	}

	infomsg := func(newname string) {
		var info string

		newname, origname = tview.Escape(newname), tview.Escape(origname)

		switch key {
		case 'M':
			info = "Created '" + newname + "' in " + selPane.getPath()

		case 'R':
			info = "Renamed '" + origname + "' to '" + newname + "'"
		}

		msgchan <- message{info, false}
	}

	input := getStatusInput(title, false)
	if rename {
		input.SetText(origname)
	}

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

func (p *dirPane) showSortDirInput() {
	input := getStatusInput("", true)

	sortmethods := []string{
		"asc",
		"desc",
		"filetype",
		"date",
		"name",
	}

	inputlabel := func() {
		label := "[::b]Sort by: "
		sortmethod, arrange := p.getSortMethod()

		for _, st := range sortmethods {
			if st == sortmethod || st == arrange {
				label += "*"
			}

			if st == "date" {
				st = "da(t)e"
			} else {
				st = "(" + string(st[0]) + ")" + string(st[1:])
			}

			label += st + " "
		}

		input.SetLabel(label)
	}

	setsort := func(t rune) {
		var s, a string

		switch t {
		case 'a':
			a = sortmethods[0]

		case 'd':
			a = sortmethods[1]

		case 'f':
			s = sortmethods[2]

		case 't':
			s = sortmethods[3]

		case 'n':
			s = sortmethods[4]
		}

		p.setSortMethod(s, a)
		inputlabel()

		p.ChangeDir(false, false)
	}

	inputlabel()

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'a', 'd', 'f', 't', 'n':
			setsort(event.Rune())
			return nil
		}

		p.table.InputHandler()(event, nil)
		statuspgs.SwitchToPage("statusmsg")
		app.SetFocus(p.table)

		return event
	})

	statuspgs.AddAndSwitchToPage("sortinput", input, true)
	app.SetFocus(input)
}

func (p *dirPane) showChangeDirInput() {
	input := getStatusInput("Change Directory to:", false)
	input.SetText(p.path)

	changeDirSelect(p, input)

	statuspgs.AddAndSwitchToPage("cdinput", input, true)
	app.SetFocus(input)
}

func showEditSelections(sinput *tview.InputField) {
	input := getStatusInput("Filter selections:", false)

	input = editSelections(input, sinput)
	if input == nil {
		return
	}

	statuspgs.AddAndSwitchToPage("editsel", input, true)
	app.SetFocus(input)
}

func execCommand() {
	imode := "Local"
	emode := "Foreground"

	input := getStatusInput("", false)

	exit := func() {
		statuspgs.SwitchToPage("statusmsg")
		app.SetFocus(prevPane.table)
	}

	inputlabel := func() {
		label := fmt.Sprintf("[::b]Exec (%s, %s): ", imode, emode)
		input.SetLabel(label)
	}

	cmdexec := func(cmdtext string) {
		if cmdtext == "" {
			return
		}

		_, err := execCmd(cmdtext, emode, imode)
		if err != nil {
			showErrorMsg(err, false)
		}
	}

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlA:
			if imode == "Local" {
				imode = "Adb"
			} else {
				imode = "Local"
			}

			inputlabel()

		case tcell.KeyCtrlQ:
			if emode == "Foreground" {
				emode = "Background"
			} else {
				emode = "Foreground"
			}

			inputlabel()

		case tcell.KeyEnter:
			cmdexec(input.GetText())
			fallthrough

		case tcell.KeyEscape:
			exit()
		}

		return event
	})

	inputlabel()

	statuspgs.AddAndSwitchToPage("exec", input, true)
	app.SetFocus(input)
}
