package main

import (
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func showHelpModal() {
	helpview := tview.NewTextView()
	helpview.SetBackgroundColor(tcell.ColorGrey)

	helpview.SetText(`
	MAIN PAGE
	=========
	Operation                     Key
	---------                     ---
	Switch between panes          Tab 
	Navigate between entries      Up, Down
	CD highlighted entry          Enter, Right
	Change one directory back     Backspace, Left
	Switch to operations page     o
	Switch between ADB/Local      s, <
	Change to any directory       g, >
	Toggle hidden files           h, .
	Refresh                       r
	Move                          m
	Paste/Put                     p
	Delete                        d
	Make directory                M
	Rename files/folders          R
	Filter entries                /
	Clear filtered entries        Ctrl+r
	Select one item               ,
	Invert selection              a
	Select all items              A
	Edit selection list           S
	Toggle layouts                [
	Swap panes                    ]
	Reset selections              Esc
	Quit                          q

	OPERATIONS PAGE
	===============
	Operation                     Key
	---------                     ---
	Navigate between entries      Up, Down
	Cancel selected operation     x
	Cancel all operations         X
	Clear operations list         C
	Switch to main page           o, Esc

	CHANGE DIRECTORY INPUT
	======================
	Operation                     Key
	---------                     ---
	Navigate between entries      Up, Down
	Autocomplete                  Tab, Any key
	CD to highlighted entry       Enter
	Move back a directory         Ctrl+W
	Switch to main page           Esc

	MKDIR/RENAME INPUT
	==================
	Operation                     Key
	---------                     ---
	Mkdir/Rename                  Enter
	Switch to main page           Esc

	DIALOG BOXES
	============
	Operation                     Key
	---------                     ---
	Switch b/w textview, buttons  Left, Right
	Scroll in textview            Up, Down
	Select highlighted button     Enter

	EDIT SELECTION DIALOG
	=====================
	Operation                     Key
	---------                     ---
	Select one item               ,
	Invert selection              a
	Select all items              A
	Switch to input               /
	Switch b/w input, list        Tab
	Save edited list              Ctrl+s
	Cancel editing list           Esc
	`)

	okbtn := tview.NewButton("Ok")
	okbtn.SetBackgroundColor(tcell.ColorBlack)

	helpview.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRight:
			app.SetFocus(okbtn)
		}

		return event
	})

	okbtn.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			app.SetFocus(helpview)

		case tcell.KeyEnter:
			pages.SwitchToPage("main")
			app.SetFocus(prevPane.table)
			prevPane.table.SetSelectable(true, false)
		}

		return event
	})

	help := modal(helpview, okbtn, nil, tcell.ColorGrey, 50, 24)
	pages.AddAndSwitchToPage("modal", help, true).ShowPage("main")

	app.SetFocus(okbtn)
}

//gocyclo:ignore
func changeDirSelect(pane *dirPane, input *tview.InputField) {
	var entries []string
	var cdfilter, cdrefresh bool

	cdtable := tview.NewTable()

	flex := tview.NewFlex().
		AddItem(cdtable, 0, 10, false).
		SetDirection(tview.FlexRow)

	infomsg := func(cdpath string) {
		if pane.path == cdpath {
			return
		}

		showInfoMsg("Changing directory to " + cdpath)
	}

	autocompletefunc := func(current string, refresh bool) {
		var row int

		if len(current) == 0 {
			return
		}

		cdfilter = true

		switch pane.mode {
		case mAdb:
			entries, _ = pane.adbListDir(current, true)
		case mLocal:
			entries, _ = pane.localListDir(current, true)
		}

		if entries == nil {
			r, _ := cdtable.GetSelection()
			cdtable.Select(r, 0)

			return
		}

		cdtable.Clear()

		for _, entry := range entries {
			if strings.Index(entry, current) != -1 {
				cdtable.SetCell(row, 0, tview.NewTableCell("[::b]"+entry).
					SetTextColor(tcell.ColorSteelBlue))

				row++
			}
		}

		cdrefresh = refresh

		cdtable.Select(0, 0)
		cdtable.ScrollToBeginning()

		cdrefresh = false
	}

	filter := func(current string) {
		var row int

		if entries == nil {
			return
		}

		cdtable.Clear()

		for _, entry := range entries {
			if strings.HasPrefix(entry, current) {
				cdtable.SetCell(row, 0, tview.NewTableCell("[::b]"+entry).
					SetTextColor(tcell.ColorSteelBlue))

				row++
			}
		}

		if row == 0 {
			pages.HidePage("modal")
		} else {
			if pg, _ := pages.GetFrontPage(); pg != "modal" {
				pages.SwitchToPage("modal").ShowPage("main")
			}
		}
		app.SetFocus(input)

		cdrefresh = true

		cdtable.Select(0, 0)
		cdtable.ScrollToBeginning()

		cdrefresh = false
	}

	input.SetChangedFunc(func(text string) {
		if cdfilter {
			return
		}

		filter(text)

		if cdtable.GetRowCount() == 0 {
			autocompletefunc(text, true)
		}
	})

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			autocompletefunc(input.GetText(), false)

		case tcell.KeyEnter:
			infomsg(input.GetText())
			pane.ChangeDir(false, false, input.GetText())
			fallthrough

		case tcell.KeyEscape:
			pages.SwitchToPage("main")
			statuspgs.SwitchToPage("statusmsg")
			app.SetFocus(pane.table)

		case tcell.KeyBackspace, tcell.KeyBackspace2:
			text := input.GetText()
			filter(input.GetText())
			autocompletefunc(filepath.Dir(text), true)

		case tcell.KeyCtrlW:
			text := trimPath(input.GetText(), true)
			input.SetText(text)
			autocompletefunc(text, true)
			return nil

		case tcell.KeyDown, tcell.KeyUp:
			cdfilter = true
			fallthrough

		case tcell.KeyPgDn, tcell.KeyPgUp:
			cdtable.InputHandler()(event, nil)
			return nil
		}

		switch event.Rune() {
		default:
			cdfilter = false
		}

		return event
	})

	cdtable.SetSelectionChangedFunc(func(row, _ int) {
		if cdrefresh || row < 0 {
			return
		}

		cell := cdtable.GetCell(row, 0)
		if cell == nil {
			return
		}

		if cell.Text == "" {
			return
		}

		input.SetText(strings.TrimPrefix(cell.Text, "[::b]"))

		cdtable.SetSelectedStyle(tcell.Style{}.
			Bold(true).
			Underline(true).
			Background(cell.Color).
			Foreground(tcell.ColorLightGrey))
	})

	autocompletefunc(pane.getPath(), false)

	cdtable.Select(0, 0)
	cdtable.SetSelectable(true, false)
	cdtable.SetBackgroundColor(tcell.ColorLightGrey)

	pages.AddAndSwitchToPage("modal", field(flex, 56, 10), true).ShowPage("main")
}

//gocyclo:ignore
func editSelections(input, sinput *tview.InputField) *tview.InputField {
	if len(multiselection) == 0 {
		return nil
	}

	var row, width int

	empty := struct{}{}
	delpaths := make(map[string]struct{}, len(multiselection))

	seltable := tview.NewTable()

	flex := tview.NewFlex().
		AddItem(seltable, 0, 10, false).
		SetDirection(tview.FlexRow)

	reset := func(p tview.Primitive, spage string) {
		app.SetFocus(p)
		statuspgs.SwitchToPage(spage)

		if spage == "confirm" {
			return
		}

		prevPane.table.SetSelectable(true, false)
	}

	exit := func() {
		pages.SwitchToPage("main")

		sel := len(multiselection) != 0

		switch {
		case sinput != nil && sel:
			reset(sinput, "confirm")

		case sinput != nil && !sel:
			fallthrough

		default:
			reset(prevPane.table, "statusmsg")
		}
	}

	save := func() {
		selectLock.Lock()
		for key := range delpaths {
			delete(multiselection, key)
		}
		selectLock.Unlock()

		delpaths = nil

		prevPane.reselect(true)
		exit()
	}

	seltoggle := func(a, i bool) {
		var color tcell.Color

		pos, _ := seltable.GetSelection()
		totalrows := seltable.GetRowCount()

		one := !a && !i
		inv := !a && i

		for row := 0; row < totalrows; row++ {
			if one {
				row, _ = seltable.GetSelection()
			}

			cell := seltable.GetCell(row, 0)

			selpath := strings.TrimPrefix(cell.Text, "[::b]")

			_, ok := delpaths[selpath]

			if !ok && (one || inv) {
				color = tcell.ColorSteelBlue
				delpaths[selpath] = empty
			} else {
				color = tcell.ColorOrangeRed
				delete(delpaths, selpath)
			}

			seltable.SetCell(row, 0, cell.SetTextColor(color))

			if one {
				if row+1 < totalrows {
					seltable.Select(row+1, 0)
					return
				}

				break
			}
		}

		seltable.Select(pos, 0)
	}

	markselected := func(i int, name string) {
		var color tcell.Color

		_, ok := delpaths[name]

		if !ok {
			color = tcell.ColorOrangeRed
		} else {
			color = tcell.ColorSteelBlue
		}

		seltable.SetCell(i, 0, tview.NewTableCell("[::b]"+name).SetTextColor(color))
	}

	input.SetChangedFunc(func(text string) {
		row := 0

		if text == "" {
			for dpath := range multiselection {
				markselected(row, dpath)
				row++
			}

			seltable.Select(0, 0)
			seltable.ScrollToBeginning()

			if pg, _ := pages.GetFrontPage(); pg != "modal" {
				pages.SwitchToPage("modal").ShowPage("main")
				app.SetFocus(input)
			}

			return
		}

		seltable.Clear()

		for dpath := range multiselection {
			if strings.Contains(
				strings.ToLower(dpath),
				strings.ToLower(text),
			) {
				markselected(row, dpath)
				row++
			}
		}

		if row == 0 {
			pages.HidePage("modal")
		} else {
			if pg, _ := pages.GetFrontPage(); pg != "modal" {
				pages.SwitchToPage("modal").ShowPage("main")
			}
		}

		app.SetFocus(input)

		seltable.Select(0, 0)
		seltable.ScrollToBeginning()
	})

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		default:
			seltable.InputHandler()(event, nil)

			if event.Modifiers() == tcell.ModAlt {
				return nil
			}
		}

		return event
	})

	seltable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			exit()

		case tcell.KeyCtrlS:
			save()
		}

		if event.Modifiers() != tcell.ModAlt {
			return event
		}

		switch event.Rune() {
		case ',':
			seltoggle(false, false)

		case 'a':
			seltoggle(false, true)

		case 'A':
			seltoggle(true, false)
		}

		return event
	})

	seltable.SetSelectionChangedFunc(func(row, col int) {
		rows := seltable.GetRowCount()

		if row < 0 || row > rows {
			return
		}

		cell := seltable.GetCell(row, col)

		if cell == nil {
			return
		}

		seltable.SetSelectedStyle(tcell.Style{}.
			Bold(true).
			Underline(true).
			Background(cell.Color).
			Foreground(tcell.ColorLightGrey))
	})

	selectLock.RLock()
	for spath := range multiselection {
		pathlen := len(spath)

		if pathlen > width {
			width = pathlen
		}

		seltable.SetCell(row, 0, tview.NewTableCell("[::b]"+spath).
			SetTextColor(tcell.ColorOrangeRed))

		row++
	}
	selectLock.RUnlock()

	if width < 50 {
		width = 50
	}

	seltable.Select(0, 0)
	seltable.SetSelectable(true, false)
	seltable.SetBackgroundColor(tcell.ColorLightGrey)

	pages.AddAndSwitchToPage("modal", field(flex, width+6, 10), true).ShowPage("main")

	return input
}

func field(v tview.Primitive, width, height int) tview.Primitive {
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(v, height, 1, false).
		AddItem(nil, 1, 1, false).
		SetDirection(tview.FlexRow)

	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(modal, width+2, 1, false).
		AddItem(nil, 0, 1, false)

	app.SetBeforeDrawFunc(func(t tcell.Screen) bool {
		width, _ := t.Size()
		flex.ResizeItem(modal, width, 0)

		return false
	})

	return flex
}

func modal(v, b, c tview.Primitive, color tcell.Color, width, height int) tview.Primitive {
	var title string

	items := tview.NewFlex()

	items.AddItem(nil, 0, 1, false)
	items.AddItem(v, height, 1, false)

	if c != nil {
		items.AddItem(c, 1, 1, false)
	}

	items.AddItem(b, 1, 1, false)
	items.AddItem(nil, 0, 1, false)
	items.SetDirection(tview.FlexRow)

	switch color {
	case tcell.ColorGrey:
		title = "[ HELP ]"
		height = height + 3

	case tcell.ColorBlack:
		title = "[ ERROR ]"
		height = height + 3

	default:
		title = "[ INFO ]"
		height = height + 4
	}

	items.SetBorder(true)
	items.SetTitle(title)
	items.SetBackgroundColor(color)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(items, height, 1, false).
		AddItem(nil, 0, 1, false).
		SetDirection(tview.FlexRow)

	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(modal, width+2, 1, false).
		AddItem(nil, 0, 1, false)
}
