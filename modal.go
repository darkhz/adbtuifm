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
	Execute command               !
	Refresh                       r
	Move                          m
	Paste/Put                     p
	Delete                        d
	Open files                    Ctrl+o
	Make directory                M
	Rename files/folders          R
	Filter entries                /
	Clear filtered entries        Ctrl+r
	Select one item               Space
	Invert selection              a
	Select all items              A
	Edit selection list           S
	Toggle layouts                [
	Swap panes                    ]
	Reset selections              Esc
	Temporarily exit to shell     Ctrl+d
	Quit                          q

	OPERATIONS PAGE
	===============
	Operation                     Key
	---------                     ---
	Navigate between entries      Up, Down
	Cancel selected operation     x
	Cancel all operations         X
	Switch to main page           o, Esc

	CHANGE DIRECTORY INPUT
	======================
	Operation                     Key
	---------                     ---
	Navigate between entries      Up, Down
	Autocomplete                  Tab
	CD to highlighted entry       Enter
	Move back a directory         Ctrl+w
	Switch to main page           Esc

	EDIT SELECTION DIALOG
	=====================
	Operation                     Key
	---------                     ---
	Select one item               Alt+Space
	Invert selection              Alt+a
	Select all items              Alt+A
	Save edited list              Ctrl+s
	Cancel editing list           Esc

	EXECUTION MODE
	==============
	Operation                     Key
	---------                     ---
	Switch b/w Local/Adb          Ctrl+a
	Switch b/w FG/BG execution    Ctrl+q
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

	help := centermodal(helpview, okbtn, "[ HELP ]", 50, 24)
	pages.AddAndSwitchToPage("centermodal", help, true).ShowPage("main")

	app.SetFocus(okbtn)
}

//gocyclo:ignore
func changeDirSelect(pane *dirPane, input *tview.InputField) {
	var modal *tview.Flex
	var cdfilter, cdrefresh bool
	var entries, entrycache []string

	dirpath := filepath.Dir(pane.getPath())

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

	reload := func(current string, refresh bool) {
		var row int
		var tmpentries []string

		if entries != nil {
			tmpentries = entries
		} else {
			tmpentries = entrycache
		}

		cdtable.Clear()

		for _, entry := range tmpentries {
			if strings.Index(entry, current) != -1 {
				cdtable.SetCell(row, 0, tview.NewTableCell("[::b]"+entry).
					SetTextColor(tcell.ColorSteelBlue))

				row++
			}
		}

		if row == 0 {
			pages.HidePage("cdmodal")
		} else {
			if pg, _ := pages.GetFrontPage(); pg != "cdmodal" {
				pages.SwitchToPage("cdmodal").ShowPage("main")
			}

			resizemodal(flex, modal, cdtable)
		}

		app.SetFocus(input)

		cdrefresh = refresh

		cdtable.Select(0, 0)
		cdtable.ScrollToBeginning()

		cdrefresh = false

		dirpath = trimPath(filepath.Dir(current), false)
	}

	autocompletefunc := func(current string, refresh bool) {
		var ok bool

		if len(current) == 0 {
			return
		}

		cdfilter = true

		switch pane.mode {
		case mAdb:
			entries, ok = pane.adbListDir(current, true)

		case mLocal:
			entries, ok = pane.localListDir(current, true)
		}

		if !ok {
			reload(current, refresh)
			return
		}

		if entries == nil {
			r, _ := cdtable.GetSelection()
			cdtable.Select(r, 0)

			return
		}

		entrycache = entries

		reload(current, refresh)
	}

	filter := func(current string) {
		if dirpath == current {
			autocompletefunc(current, true)
		} else {
			reload(current, true)
		}
	}

	input.SetChangedFunc(func(text string) {
		if cdfilter {
			return
		}

		if text == "" {
			input.SetText("/")
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

	cdtable.Select(0, 0)
	cdtable.SetSelectable(true, false)
	cdtable.SetBackgroundColor(tcell.ColorLightGrey)

	view, modal := statusmodal(flex)
	pages.AddPage("cdmodal", view, true, false).ShowPage("main")

	autocompletefunc(pane.getPath(), false)
}

//gocyclo:ignore
func editSelections(input, sinput *tview.InputField) *tview.InputField {
	if len(multiselection) == 0 {
		return nil
	}

	var row int
	var modal *tview.Flex

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

			if pg, _ := pages.GetFrontPage(); pg != "editmodal" {
				pages.SwitchToPage("editmodal").ShowPage("main")
				app.SetFocus(input)
			}

			resizemodal(flex, modal, seltable)

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
			pages.HidePage("editmodal")
		} else {
			if pg, _ := pages.GetFrontPage(); pg != "editmodal" {
				pages.SwitchToPage("editmodal").ShowPage("main")
			}

			resizemodal(flex, modal, seltable)
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
		case ' ':
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
		seltable.SetCell(row, 0, tview.NewTableCell("[::b]"+spath).
			SetTextColor(tcell.ColorOrangeRed))

		row++
	}
	selectLock.RUnlock()

	seltable.Select(0, 0)
	seltable.SetSelectable(true, false)
	seltable.SetBackgroundColor(tcell.ColorLightGrey)

	view, modal := statusmodal(flex)
	resizemodal(flex, modal, seltable)
	pages.AddAndSwitchToPage("editmodal", view, true).ShowPage("main")

	return input
}

func resizemodal(f, m *tview.Flex, t *tview.Table) {
	height := t.GetRowCount()

	_, _, _, screenHeight := pages.GetRect()
	screenHeight /= 4

	if height > screenHeight {
		height = screenHeight
	}

	f.ResizeItem(t, height, 0)
	m.ResizeItem(f, height, 0)
}

func statusmodal(v tview.Primitive) (tview.Primitive, *tview.Flex) {
	_, _, _, screenHeight := pages.GetRect()
	screenHeight /= 4

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(v, screenHeight, 1, false).
		AddItem(nil, 1, 1, false).
		SetDirection(tview.FlexRow)

	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(modal, 10, 1, false).
		AddItem(nil, 0, 1, false)

	app.SetBeforeDrawFunc(func(t tcell.Screen) bool {
		width, _ := t.Size()
		flex.ResizeItem(modal, width, 0)

		return false
	})

	return flex, modal
}

func centermodal(v, b tview.Primitive, title string, width, height int) tview.Primitive {
	items := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(v, height, 1, false).
		AddItem(b, 1, 1, false).
		AddItem(nil, 0, 1, false).
		SetDirection(tview.FlexRow)

	height += 3

	items.SetBorder(true)
	items.SetTitle(title)

	centermodal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(items, height, 1, false).
		AddItem(nil, 0, 1, false).
		SetDirection(tview.FlexRow)

	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(centermodal, width+2, 1, false).
		AddItem(nil, 0, 1, false)
}
