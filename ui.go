package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/sync/semaphore"
)

var (
	app      *tview.Application
	pages    *tview.Pages
	opsView  *tview.Table
	prevPane *dirPane

	entrycache []string
)

func setupUI() {
	app = tview.NewApplication()
	pages = tview.NewPages()

	pages.AddPage("main", setupPaneView(), true, true)
	pages.AddPage("ops", setupInfoView(), true, true)

	pages.SwitchToPage("main")

	if err := app.SetRoot(pages, true).SetFocus(prevPane.tbl).Run(); err != nil {
		panic(err)
	}
}

func setupPaneView() *tview.Flex {
	selPane := &dirPane{0, semaphore.NewWeighted(1), tview.NewTable(), initMode, initPath, initAPath, initLPath, true, false, nil}
	auxPane := &dirPane{0, semaphore.NewWeighted(1), tview.NewTable(), initMode, initPath, initAPath, initLPath, true, false, nil}

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
	if opsView != nil {
		goto HEADER
	}

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
		case 'C':
			clearAllOps()
		case 'X':
			go cancelAllOps()
		}

		return event
	})

HEADER:
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
		selPane.updatePrevPane()

		switch event.Key() {
		case tcell.KeyEscape:
			reset(selPane, auxPane)
		case tcell.KeyTab:
			selected(selPane, auxPane)
		case tcell.KeyEnter:
			selPane.ChangeDir(true, false)
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			selPane.ChangeDir(false, true)
		}

		switch event.Rune() {
		case 'm', 'c', 'p', 'd':
			opsHandler(selPane, auxPane, event.Rune())
		case 's':
			modeSwitchHandler(selPane)
		case 'S':
			selPane.multiSelectHandler(false)
		case 'A':
			selPane.multiSelectHandler(true)
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

		switch {
		case tmpentry != nil:
			entrycache = tmpentry
		case tmpentry == nil && entrycache != nil:
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
		case tcell.KeyEscape:
			pages.SwitchToPage("main")
			app.SetFocus(p.tbl)
		case tcell.KeyEnter:
			pages.SwitchToPage("main")
			app.SetFocus(p.tbl)
			p.ChangeDir(false, false, input.GetText())
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

	pages.AddAndSwitchToPage("modal", modal(input, 80, 3), true).ShowPage("main")
	app.SetFocus(input)
}

func showConfirmModal(msg string, alert bool, dofunc, resetfunc func()) {
	view := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	okbtn := tview.NewButton("Ok")
	cancelbtn := tview.NewButton("Cancel")

	okbtn.SetBackgroundColor(tcell.ColorBlack)
	cancelbtn.SetBackgroundColor(tcell.ColorBlack)

	if alert {
		view.SetBackgroundColor(tcell.ColorRed)
	} else {
		view.SetBackgroundColor(tcell.ColorBlue)
	}

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
			resetfunc()
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

	pages.AddAndSwitchToPage("modal", infomodal(view, okbtn, cancelbtn, alert, 50, 10), true).ShowPage("main")
	app.SetFocus(cancelbtn)

}

func showErrorModal(msg string) {
	errview := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

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
		case tcell.KeyEnter:
			pages.SwitchToPage("main")
			app.SetFocus(prevPane.tbl)
		case tcell.KeyLeft:
			app.SetFocus(errview)
		}

		return event
	})

	msg = fmt.Sprintf("[red]An error has occurred:\n\n%s", msg)
	errview.SetText(msg)

	pages.AddAndSwitchToPage("modal", errmodal(errview, okbtn, 50, 10), true).ShowPage("main")
	app.SetFocus(okbtn)
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

func infomodal(p, b, c tview.Primitive, alert bool, width, height int) tview.Primitive {
	flex := tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, false).
			AddItem(c, 1, 1, false).
			AddItem(b, 1, 1, false).
			AddItem(nil, 0, 1, false), width, 1, false)

	flex.SetBorder(true)
	flex.SetTitle("| INFO |")

	if alert {
		flex.SetBackgroundColor(tcell.ColorRed)
	} else {
		flex.SetBackgroundColor(tcell.ColorBlue)
	}

	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(flex, height+4, 1, false).
			AddItem(nil, 0, 1, false), width+2, 1, false).
		AddItem(nil, 0, 1, false)
}

func reset(selPane, auxPane *dirPane) {
	ops = opNone
	srcPaths = nil
	selstart = false

	selPane.selected = false
	auxPane.selected = false

	setOpsLock(false)
	selPane.setPaneOpStatus(false)

	app.SetFocus(selPane.tbl)
	selPane.tbl.SetSelectable(true, false)
	auxPane.tbl.SetSelectable(false, false)

	selPane.resetSelection()
	auxPane.resetSelection()
}

func selected(selPane, auxPane *dirPane) {
	if !getOpsLock() && !selstart {
		selPane.tbl.SetSelectable(false, false)
		auxPane.tbl.SetSelectable(true, false)
		app.SetFocus(auxPane.tbl)
	}
}

func (p *dirPane) gotoOpsPage() {
	setProgress(true)
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

func (p *dirPane) updateDirPane(row int, sel bool, cell *tview.TableCell, name ...string) {
	var tblcell *tview.TableCell

	color := tcell.ColorSkyblue

	if sel {
		color = tcell.ColorOrange
	}

	if cell != nil {
		tblcell = cell
	} else {
		tblcell = tview.NewTableCell(name[0])
	}

	p.tbl.SetCell(row, 0, tblcell.SetTextColor(color))
}

func (p *dirPane) resetSelection() {
	go func() {
		if !p.getLock() {
			return
		}
		defer p.setUnlock()

		app.QueueUpdateDraw(func() {
			for i := 0; i < p.tbl.GetRowCount(); i++ {
				cell := p.tbl.GetCell(i, 0)

				p.updateDirPane(i, false, cell)
			}
		})
	}()
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
		prevPane.tbl.SetSelectable(true, false)
		return
	}

	app.QueueUpdateDraw(func() {
		p.tbl.SetSelectable(false, false)
	})
}

func (p *dirPane) setHidden() {
	if !p.getLock() {
		return
	}
	defer p.setUnlock()

	switch p.hidden {
	case true:
		p.hidden = false
	case false:
		p.hidden = true
	}

	p.ChangeDir(false, false)
}

func (p *dirPane) getHidden() bool {
	return p.hidden
}

func stopApp() {
	showConfirmModal("Do you want to quit?", false, func() {
		app.Stop()
		cancelAllOps()
	}, func() { app.SetFocus(prevPane.tbl) })
}
