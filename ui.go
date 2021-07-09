package main

import (
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	app      *tview.Application
	pages    *tview.Pages
	opsView  *tview.Table
	prevPane *dirPane
)

func setupUI() {
	app = tview.NewApplication()
	pages = tview.NewPages()

	pages.AddPage("main", setupPaneView(), true, true)
	pages.AddPage("ops", setupInfoView(), true, true)

	pages.SwitchToPage("main")
	app.SetFocus(prevPane.tbl)

	if err := app.SetRoot(pages, true).SetFocus(prevPane.tbl).Run(); err != nil {
		panic(err)
	}
}

func setupPaneView() *tview.Flex {
	var leftpane = dirPane{tview.NewTable(), 0, initMode, initPath, initAPath, initLPath, nil}
	var rightpane = dirPane{tview.NewTable(), 0, initMode, initPath, initAPath, initLPath, nil}

	selPane := &leftpane
	auxPane := &rightpane

	prevPane = selPane

	setupPane(selPane, auxPane)
	setupPane(auxPane, selPane)

	flex := tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(leftpane.tbl, 0, 1, true).
				AddItem(rightpane.tbl, 0, 1, false), 0, 2, true), 0, 2, false)

	flex.SetBorder(true)
	flex.SetTitle("| ADBTuiFM |")

	flex.SetBackgroundColor(tcell.Color16)
	selPane.tbl.SetBackgroundColor(tcell.Color16)
	auxPane.tbl.SetBackgroundColor(tcell.Color16)

	selPane.ChangeDir(false, false)
	auxPane.ChangeDir(false, false)

	return flex
}

func setupInfoView() *tview.Table {
	opsView = tview.NewTable()

	opsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			pages.SwitchToPage("main")
			app.SetFocus(prevPane.tbl)
			opsView.SetSelectable(false, false)
		}

		switch event.Rune() {
		case 'o':
			pages.SwitchToPage("main")
			app.SetFocus(prevPane.tbl)
			opsView.SetSelectable(false, false)
		case 'q':
			stopApp()
		case 'x':
			r, _ := opsView.GetSelection()
			id, _ := strconv.Atoi(opsView.GetCell(r, 0).Text)
			cancelOps(id)
		case 'X':
			go cancelAllOps()
		}

		return event
	})

	opsView.SetTitle("Operations")
	opsView.SetFixed(1, 1)
	opsView.SetCell(0, 0, tview.NewTableCell("ID").
		SetSelectable(false))
	opsView.SetCell(0, 1, tview.NewTableCell("Type").
		SetExpansion(1).
		SetAlign(tview.AlignCenter).
		SetSelectable(false))
	opsView.SetCell(0, 2, tview.NewTableCell("Path").
		SetExpansion(1).
		SetAlign(tview.AlignCenter).
		SetSelectable(false))
	opsView.SetCell(0, 3, tview.NewTableCell("Status").
		SetExpansion(1).
		SetAlign(tview.AlignCenter).
		SetSelectable(false))
	opsView.SetBorder(true)
	opsView.SetBorders(true)
	opsView.SetSelectable(false, false)

	opsView.SetBackgroundColor(tcell.Color16)

	return opsView
}

func setupPane(selPane *dirPane, auxPane *dirPane) {
	selPane.ChangeDir(false, false)

	selPane.tbl.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		selPane.row, _ = selPane.tbl.GetSelection()

		switch event.Key() {
		case tcell.KeyEscape:
			ops = opNone
			opsLock = false
			app.SetFocus(auxPane.tbl)
		case tcell.KeyTab:
			if !opsLock {
				selPane.tbl.SetSelectable(false, false)
				auxPane.tbl.SetSelectable(true, false)
				app.SetFocus(auxPane.tbl)
			}
		case tcell.KeyEnter:
			selPane.tbl.SetSelectable(true, true)
			selPane.tbl.SetSelectedFunc(func(row int, column int) {
				selPane.ChangeDir(true, false)
			})
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			selPane.ChangeDir(false, true)
		}

		switch event.Rune() {
		case 'm', 'c', 'p', 'd':
			prevPane = selPane
			opsHandler(selPane, auxPane, event.Rune())
		case 's':
			modeSwitchHandler(selPane)
			selPane.ChangeDir(false, false)
		case 'r':
			selPane.ChangeDir(false, false)
		case 'o':
			prevPane = selPane
			pages.SwitchToPage("ops")
			app.SetFocus(opsView)
			opsView.SetSelectable(true, false)
		case 'h':
			if setHidden == false {
				setHidden = true
			} else {
				setHidden = false
			}
			selPane.ChangeDir(false, false)
			auxPane.ChangeDir(false, false)
		case 'g':
			selPane.showChangeDirInput()
		case 'q':
			stopApp()
		}

		return event
	})

	selPane.tbl.SetBorder(true)
	selPane.tbl.SetSelectable(true, true)
}

func showErrorModal(msg string) {
	msgbox := tview.NewModal().
		SetText("[red]ERROR: " + msg).
		AddButtons([]string{"Ok"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.SwitchToPage("main")
			app.SetFocus(prevPane.tbl)
		})

	pages.AddAndSwitchToPage("modal", modal(msgbox, 80, 29), true).ShowPage("main")
	app.SetFocus(msgbox)
}

func showConfirmModal(msg string, Dofunc func()) {
	msgbox := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"Ok", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Ok" {
				Dofunc()
			}
			pages.SwitchToPage("main")
			app.SetFocus(prevPane.tbl)
		})

	pages.AddAndSwitchToPage("modal", modal(msgbox, 80, 29), true).ShowPage("main")
	app.SetFocus(msgbox)
}

func (p *dirPane) showChangeDirInput() {
	input := tview.NewInputField()
	input.SetTitle("Change Directory to:")
	input.SetTitleAlign(tview.AlignCenter)
	input.SetBorder(true)
	input.SetText(p.path)

	input.SetAutocompleteFunc(func(current string) (entries []string) {
		var iserr bool

		if len(current) == 0 {
			return
		}

		switch p.mode {
		case mAdb:
			entries, iserr = p.adbListDir(current, true)
		case mLocal:
			entries, iserr = p.localListDir(current, true)
		}

		if !iserr {
			return
		}

		return
	})

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			pages.SwitchToPage("main")
			app.SetFocus(p.tbl)
		case tcell.KeyEnter:
			pages.SwitchToPage("main")
			app.SetFocus(p.tbl)

			switch p.mode {
			case mAdb:
				p.apath = p.path
			case mLocal:
				p.dpath = p.path
			}

			p.path = trimPath(input.GetText(), false)
			p.ChangeDir(false, false)
		}

		return event
	})

	pages.AddAndSwitchToPage("modal", modal(input, 80, 3), true).ShowPage("main")
	app.SetFocus(input)
}

func modal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}

func setPaneTitle(pane *dirPane) {
	prefix := ""

	switch pane.mode {
	case mAdb:
		prefix = "Adb: "
	case mLocal:
		prefix = "Local: "
	}

	if pane.path == "./" || pane.path == "../" {
		pane.path = "/"
	}

	pane.tbl.SetTitle("|- " + prefix + pane.path + " -|")
}

func (p *dirPane) updateDirPane(row int, name string) {
	p.tbl.SetCell(row, 0, tview.NewTableCell(name).
		SetTextColor(tcell.ColorSkyblue))
}

func stopApp() {
	showConfirmModal("Do you want to quit?", func() {
		app.Stop()
		cancelAllOps()
	})
}
