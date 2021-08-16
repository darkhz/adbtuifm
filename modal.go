package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sahilm/fuzzy"
)

var (
	mrinput    string
	entrycache []string
)

func showMkdirRenameInput(selPane, auxPane *dirPane, key rune) {
	var title string

	switch key {
	case 'M':
		title = "Make directory"
	case 'R':
		title = "Rename To"
	}

	input := tview.NewInputField()

	input.SetBorder(true)
	input.SetTitle(title)
	input.SetTitleAlign(tview.AlignCenter)

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			text := input.GetText()
			if text != "" {
				mrinput = text
				opsHandler(selPane, auxPane, key)
			}

			fallthrough

		case tcell.KeyEscape:
			pages.SwitchToPage("main")
			app.SetFocus(selPane.table)
		}

		return event
	})

	pages.AddAndSwitchToPage("modal", field(input, 60, 3), true).ShowPage("main")
	app.SetFocus(input)
}

func (p *dirPane) showFilterInput() {
	input := tview.NewInputField()

	input.SetBorder(true)
	input.SetTitle("Filter")
	input.SetTitleAlign(tview.AlignCenter)

	markselected := func(row int, pathstr string) {
		if checkSelected(p.path, pathstr, false) {
			p.updateDirPane(row, true, nil, pathstr)
		} else {
			p.updateDirPane(row, false, nil, pathstr)
		}
	}

	contains := func(needle int, haystack []int) bool {
		for _, i := range haystack {
			if needle == i {
				return true
			}
		}
		return false
	}

	input.SetChangedFunc(func(text string) {
		if !p.getLock() {
			return
		}
		defer p.setUnlock()

		entries := make([]string, len(p.pathList))

		p.table.Clear()

		for row, dir := range p.pathList {
			if text == "" {
				markselected(row, dir.Name)
				continue
			}

			entries[row] = dir.Name
		}

		if text == "" {
			p.table.Select(0, 0)
			p.table.ScrollToBeginning()

			return
		}

		matches := fuzzy.Find(text, entries)

		for row, match := range matches {
			for i := 0; i < len(match.Str); i++ {
				if contains(i, match.MatchedIndexes) {
					markselected(row, string(match.Str))
				}
			}
		}

		p.table.Select(0, 0)
		p.table.ScrollToBeginning()
	})

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyEnter:
			pages.SwitchToPage("main")
			app.SetFocus(p.table)
		}

		return event
	})

	pages.AddAndSwitchToPage("modal", field(input, 60, 3), true).ShowPage("main")
	app.SetFocus(input)
}

func (p *dirPane) showChangeDirInput() {
	input := tview.NewInputField()

	input.SetText(p.path)
	input.SetBorder(true)
	input.SetTitle("Change Directory to:")
	input.SetTitleAlign(tview.AlignCenter)

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
			p.ChangeDir(false, false, input.GetText())
			fallthrough

		case tcell.KeyEscape:
			pages.SwitchToPage("main")
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

	pages.AddAndSwitchToPage("modal", field(input, 60, 3), true).ShowPage("main")
	app.SetFocus(input)
}

func showConfirmModal(msg string, alert bool, dofunc, resetfunc func()) {
	var color tcell.Color

	if alert {
		color = tcell.ColorRed
	} else {
		color = tcell.ColorBlue
	}

	view := tview.NewTextView()
	view.SetScrollable(true)
	view.SetBackgroundColor(color)

	okbtn := tview.NewButton("Ok")
	okbtn.SetBackgroundColor(tcell.ColorBlack)

	cancelbtn := tview.NewButton("Cancel")
	cancelbtn.SetBackgroundColor(tcell.ColorBlack)

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRight:
			app.SetFocus(cancelbtn)
		}

		return event
	})

	cancelbtn.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			pages.SwitchToPage("main")
			app.SetFocus(prevPane.table)

		case tcell.KeyLeft:
			app.SetFocus(view)

		case tcell.KeyRight:
			app.SetFocus(okbtn)

		}

		return event
	})

	okbtn.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			pages.SwitchToPage("main")
			dofunc()
			resetfunc()

		case tcell.KeyLeft:
			app.SetFocus(cancelbtn)
		}

		return event
	})

	msg = fmt.Sprintf("%s", msg)
	view.SetText(msg)

	info := modal(view, okbtn, cancelbtn, color, 50, 10)
	pages.AddAndSwitchToPage("modal", info, true).ShowPage("main")

	app.SetFocus(cancelbtn)
}

func showErrorModal(msg string) {
	errview := tview.NewTextView()
	errview.SetDynamicColors(true)
	errview.SetScrollable(true)

	okbtn := tview.NewButton("Ok")

	errview.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRight:
			app.SetFocus(okbtn)
		}

		return event
	})

	okbtn.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			app.SetFocus(errview)

		case tcell.KeyEnter:
			pages.SwitchToPage("main")
			app.SetFocus(prevPane.table)
			prevPane.table.SetSelectable(true, false)
		}

		return event
	})

	msg = fmt.Sprintf("[red]An error has occurred:\n\n%s", msg)
	errview.SetText(msg)

	err := modal(errview, okbtn, nil, tcell.ColorBlack, 50, 10)
	pages.AddAndSwitchToPage("modal", err, true).ShowPage("main")

	app.SetFocus(okbtn)
}

func showHelpModal() {
	helpview := tview.NewTextView()
	helpview.SetBackgroundColor(tcell.ColorGrey)

	helpview.SetText(`
	MAIN PAGE
	=========
	Operation                     Key
	---------                     ---
	Switch between panes          Tab 
	Navigate between entries      Up/Down
	CD highlighted entry          Enter/Right
	Change one directory back     Backspace/Left
	Switch between ADB/Local      s
	Switch to operations page     o
	Change to any directory       g
	Refresh                       r
	Move                          m
	Paste/Put                     p
	Delete                        d
	Make directory                M
	Rename files/folders          R
	Toggle hidden files           h
	Select one item               ,
	Invert selection              a
	Select all items              A
	Reset selections              Esc
	Quit                          q

	OPERATIONS PAGE
	===============
	Operation                     Key
	---------                     ---
	Navigate between entries      Up/Down
	Switch to main page           o/Esc
	Cancel selected operation     x
	Cancel all operations         X
	Clear operations list         C

	CHANGE DIRECTORY INPUT
	======================
	Operation                     Key
	---------                     ---
	Navigate between entries      Up/Down
	Autocomplete                  Tab/Any key
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
	Switch b/w textview, buttons  Left/Right
	Scroll in textview            Up/Down
	Select highlighted button     Enter

	EDIT SELECTION DIALOG
	=====================
	Operation                     Key
	---------                     ---
	Select one item               ,
	Invert selection              a
	Select all items              A
	Switch b/w input, list        /, Tab
	Save edited list              Ctrl+S
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

func showEditSelections() {
	if len(multiselection) == 0 {
		showErrorModal("Selection list empty")
		return
	}

	var focus bool
	var row, width int

	empty := struct{}{}
	delpaths := make(map[string]struct{})

	seltable := tview.NewTable()

	input := tview.NewInputField()
	input.SetLabel("Filter: ")

	flex := tview.NewFlex().
		AddItem(input, 0, 2, false).
		AddItem(seltable, 0, 10, false).
		SetDirection(tview.FlexRow)

	exit := func() {
		pages.SwitchToPage("main")
		app.SetFocus(prevPane.table)
		prevPane.table.SetSelectable(true, false)
	}

	save := func() {
		selectLock.Lock()
		for key, _ := range delpaths {
			delete(multiselection, key)
		}
		selectLock.Unlock()

		prevPane.reselect(false)
		exit()
	}

	focustoggle := func() {
		if !focus {
			focus = true
			app.SetFocus(input)
		} else {
			focus = false
			app.SetFocus(seltable)
		}
	}

	seltoggle := func(a, i bool) {
		var color tcell.Color

		totalrows := seltable.GetRowCount()

		one := !a && !i
		inv := !a && i

		for row := 0; row < totalrows; row++ {
			if one {
				row, _ = seltable.GetSelection()
			}

			cell := seltable.GetCell(row, 0)

			selpath := cell.Text

			_, ok := delpaths[selpath]

			if !ok && (one || inv) {
				color = tcell.ColorSkyblue
				delpaths[selpath] = empty
			} else {
				color = tcell.ColorOrange
				delete(delpaths, selpath)
			}

			seltable.SetCell(row, 0, cell.SetTextColor(color))

			if row+1 < totalrows && one {
				seltable.Select(row+1, 0)
			}

			if one {
				return
			}
		}
	}

	markselected := func(i int, name string) {
		var color tcell.Color

		_, ok := delpaths[name]

		if !ok {
			color = tcell.ColorOrange
		} else {
			color = tcell.ColorSkyblue
		}

		seltable.SetCell(i, 0, tview.NewTableCell(name).SetTextColor(color))
	}

	contains := func(needle int, haystack []int) bool {
		for _, i := range haystack {
			if needle == i {
				return true
			}
		}
		return false
	}

	input.SetChangedFunc(func(text string) {
		var selrow int

		entries := make([]string, len(multiselection))

		seltable.Clear()

		selectLock.RLock()
		for key, _ := range multiselection {
			if text == "" {
				markselected(selrow, key)
				selrow++
				continue
			}

			entries[selrow] = key
			selrow++
		}
		selectLock.RUnlock()

		if text == "" {
			seltable.Select(0, 0)
			seltable.ScrollToBeginning()

			return
		}

		matches := fuzzy.Find(text, entries)

		for row, match := range matches {
			for i := 0; i < len(match.Str); i++ {
				if contains(i, match.MatchedIndexes) {
					markselected(row, string(match.Str))
				}
			}
		}

		seltable.Select(0, 0)
		seltable.ScrollToBeginning()
	})

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			exit()

		case tcell.KeyCtrlS:
			save()

		case tcell.KeyTab:
			focustoggle()
			seltable.SetSelectable(true, false)
		}

		return event
	})

	seltable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			exit()

		case tcell.KeyCtrlS:
			save()

		case tcell.KeyTab:
			focustoggle()
			seltable.SetSelectable(false, false)
		}

		switch event.Rune() {
		case ',':
			seltoggle(false, false)

		case 'a':
			seltoggle(false, true)

		case 'A':
			seltoggle(true, false)

		case '/':
			focustoggle()
			seltable.SetSelectable(false, false)
		}

		return event
	})

	selectLock.RLock()
	for spath, _ := range multiselection {
		pathlen := len(spath)

		if pathlen > width {
			width = pathlen
		}

		seltable.SetCell(row, 0, tview.NewTableCell(spath).
			SetTextColor(tcell.ColorOrange))

		row++
	}
	selectLock.RUnlock()

	flex.SetBorder(true)
	seltable.SetSelectable(true, false)
	flex.SetTitle("[ EDIT SELECTION ]")

	pages.AddAndSwitchToPage("modal", field(flex, width+6, 23), true).ShowPage("main")
	app.SetFocus(seltable)
}

func field(v tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(v, 1, 1, 1, 1, 0, 0, true)
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
