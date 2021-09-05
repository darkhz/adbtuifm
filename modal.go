package main

import (
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
func showEditSelections(status *tview.InputField) {
	if len(multiselection) == 0 {
		return
	}

	var focus bool
	var row, width int

	empty := struct{}{}
	delpaths := make(map[string]struct{}, len(multiselection))

	seltable := tview.NewTable()

	input := tview.NewInputField()
	input.SetLabel("Filter: ")

	flex := tview.NewFlex().
		AddItem(input, 0, 2, false).
		AddItem(seltable, 0, 10, false).
		SetDirection(tview.FlexRow)

	exit := func() {
		pages.SwitchToPage("main")

		if status != nil {
			if len(multiselection) == 0 {
				statuspgs.SwitchToPage("statusmsg")
				app.SetFocus(prevPane.table)
				return
			}

			app.SetFocus(status)

		} else {
			app.SetFocus(prevPane.table)
			prevPane.table.SetSelectable(true, false)
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

	input.SetChangedFunc(func(text string) {
		row := 0

		if text == "" {
			for dpath := range multiselection {
				markselected(row, dpath)
				row++
			}

			seltable.Select(0, 0)
			seltable.ScrollToBeginning()

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
	for spath := range multiselection {
		pathlen := len(spath)

		if pathlen > width {
			width = pathlen
		}

		seltable.SetCell(row, 0, tview.NewTableCell(spath).
			SetTextColor(tcell.ColorOrange))

		row++
	}
	selectLock.RUnlock()

	if width < 50 {
		width = 50
	}

	seltable.SetSelectable(true, false)

	flex.SetBorder(true)
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
