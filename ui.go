package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/sync/semaphore"
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
	selPane := &dirPane{0, semaphore.NewWeighted(1), sync.Mutex{}, tview.NewTable(), initMode, initPath, initAPath, initLPath, true, false, nil}
	auxPane := &dirPane{0, semaphore.NewWeighted(1), sync.Mutex{}, tview.NewTable(), initMode, initPath, initAPath, initLPath, true, false, nil}

	prevPane = selPane

	setupPane(selPane, auxPane)
	setupPane(auxPane, selPane)

	flex := tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(selPane.tbl, 0, 1, true).
				AddItem(auxPane.tbl, 0, 1, false), 0, 2, true), 0, 2, false)

	flex.SetBorder(true)
	flex.SetTitle("| ADBTuiFM |")
	flex.SetBackgroundColor(tcell.Color16)

	selPane.tbl.SetBackgroundColor(tcell.Color16)
	auxPane.tbl.SetBackgroundColor(tcell.Color16)

	return flex
}

func setupInfoView() *tview.Table {
	opsView = tview.NewTable()

	opsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			setProgress(false)
			pages.SwitchToPage("main")
			app.SetFocus(prevPane.tbl)
			opsView.SetSelectable(false, false)
		}

		switch event.Rune() {
		case 'o':
			setProgress(false)
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

	opsView.SetFixed(1, 1)
	opsView.SetBorder(true)
	opsView.SetBorders(true)
	opsView.SetSelectable(false, false)
	opsView.SetBackgroundColor(tcell.Color16)

	opsView.SetTitle("| Operations |")

	return opsView
}

func setupPane(selPane, auxPane *dirPane) {
	selPane.tbl.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			ops = opNone
			setOpsLock(false)
			app.SetFocus(selPane.tbl)
			selPane.setPaneOpStatus(false)
		case tcell.KeyTab:
			if !getOpsLock() {
				selPane.tbl.SetSelectable(false, false)
				auxPane.tbl.SetSelectable(true, false)
				app.SetFocus(auxPane.tbl)
			}
		case tcell.KeyEnter:
			selPane.tbl.SetSelectable(true, true)
			selPane.ChangeDir(true, false)
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if selPane.getPending() {
				return nil
			}

			selPane.ChangeDir(false, true)
		}

		switch event.Rune() {
		case 'm', 'c', 'p', 'd':
			opsHandler(selPane, auxPane, event.Rune())
		case 's':
			modeSwitchHandler(selPane)
		case 'r':
			selPane.ChangeDir(false, false)
		case 'o':
			selPane.gotoOpsPage()
		case 'h':
			selPane.setHidden()
		case 'g':
			selPane.showChangeDirInput()
		case 'q':
			stopApp()
		}

		return event
	})

	selPane.tbl.SetBorder(true)
	selPane.tbl.SetSelectable(true, true)

	selPane.ChangeDir(false, false)
}

func (p *dirPane) showChangeDirInput() {
	input := tview.NewInputField()

	input.SetText(p.path)
	input.SetBorder(true)
	input.SetTitle("Change Directory to:")
	input.SetTitleAlign(tview.AlignCenter)

	p.updatePrevPane()

	input.SetAutocompleteFunc(func(current string) (entries []string) {
		if len(current) == 0 || !strings.HasSuffix(current, "/") {
			return
		}

		switch p.mode {
		case mAdb:
			entries, _ = p.adbListDir(current, true)
		case mLocal:
			entries, _ = p.localListDir(current, true)
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
			p.ChangeDir(false, false, input.GetText())
		}

		return event
	})

	pages.AddAndSwitchToPage("modal", modal(input, 80, 3), true).ShowPage("main")
	app.SetFocus(input)
}

func showErrorModal(msg string) {
	errview := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	okbtn := tview.NewButton("OK")

	errview.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyDown:
			app.SetFocus(okbtn)
		case tcell.KeyRight:
			errview.ScrollToEnd()
		}

		return event
	})

	okbtn.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			pages.SwitchToPage("main")
			app.SetFocus(prevPane.tbl)
		case tcell.KeyUp:
			app.SetFocus(errview)
		}

		return event
	})

	msg = fmt.Sprintf("[red]An error has occurred:\n\n%s", msg)
	errview.SetText(msg)

	pages.AddAndSwitchToPage("modal", errmodal(errview, okbtn, 60, 12), true).ShowPage("main")
	app.SetFocus(okbtn)
}

func showConfirmModal(msg string, alert bool, Dofunc func()) {
	msgbox := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"Cancel", "Ok"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Ok" {
				Dofunc()
			}
			pages.SwitchToPage("main")
			app.SetFocus(prevPane.tbl)
		})

	if alert {
		msgbox.SetBackgroundColor(tcell.ColorRed)
	} else {
		msgbox.SetBackgroundColor(tcell.ColorSteelBlue)
	}

	pages.AddAndSwitchToPage("modal", modal(msgbox, 80, 29), true).ShowPage("main")
	app.SetFocus(msgbox)
}

func modal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}

func errmodal(p, b tview.Primitive, width, height int) tview.Primitive {
	flex := tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, false).
			AddItem(b, 1, 1, false).
			AddItem(nil, 0, 1, false), width, 1, false)

	flex.SetBorder(true)
	flex.SetTitle("| ERROR |")

	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(flex, height+3, 1, false).
			AddItem(nil, 0, 1, false), width+2, 1, false).
		AddItem(nil, 0, 1, false)
}

func (p *dirPane) gotoOpsPage() {
	setProgress(true)
	p.updatePrevPane()
	app.SetFocus(opsView)
	pages.SwitchToPage("ops")
	opsView.SetSelectable(true, false)
}

func (p *dirPane) setPaneTitle() {
	prefix := ""

	switch p.mode {
	case mAdb:
		prefix = "Adb"
	case mLocal:
		prefix = "Local"
	}

	if p.path == "./" || p.path == "../" {
		p.path = "/"
	} else {
		p.path = trimPath(p.path, false)
	}

	title := fmt.Sprintf("|- %s: %s -|", prefix, p.path)
	p.tbl.SetTitle(title)
}

func (p *dirPane) updatePrevPane() {
	prevPane = p
}

func (p *dirPane) updateDirPane(row int, name string) {
	p.tbl.SetCell(row, 0, tview.NewTableCell(name).
		SetTextColor(tcell.ColorSkyblue))
}

func (p *dirPane) updateRow(lock bool) {
	if !lock {
		p.row, _ = p.tbl.GetSelection()
		return
	}

	app.QueueUpdateDraw(func() {
		p.row, _ = p.tbl.GetSelection()
	})
}

func (p *dirPane) setPaneOpStatus(pending bool) {
	color := tcell.ColorSteelBlue

	if !pending {
		color = tcell.ColorWhite
	}

	p.tbl.SetBorderColor(color)
}

func (p *dirPane) setPaneListStatus(pending bool) {
	if !pending {
		p.setPending(false)
		p.tbl.SetSelectable(true, false)
		return
	}

	p.setPending(true)
	app.QueueUpdateDraw(func() {
		p.tbl.SetSelectable(false, false)
	})
}

func (p *dirPane) setPending(pending bool) {
	p.plock.Lock()
	defer p.plock.Unlock()

	p.pending = pending
}

func (p *dirPane) setHidden() {
	p.plock.Lock()

	switch p.hidden {
	case true:
		p.hidden = false
	case false:
		p.hidden = true
	}

	p.plock.Unlock()

	p.ChangeDir(false, false)
}

func (p *dirPane) getPending() bool {
	p.plock.Lock()
	defer p.plock.Unlock()

	return p.pending
}

func (p *dirPane) getHidden() bool {
	p.plock.Lock()
	defer p.plock.Unlock()

	return p.hidden
}

func stopApp() {
	showConfirmModal("Do you want to quit?", false, func() {
		app.Stop()
		cancelAllOps()
	})
}
