package main

import (
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	adb "github.com/zach-klippenstein/goadb"
	"golang.org/x/sync/semaphore"
)

type dirPane struct {
	row      int
	path     string
	apath    string
	dpath    string
	hidden   bool
	mode     ifaceMode
	table    *tview.Table
	plock    *semaphore.Weighted
	pathList []*adb.DirEntry
}

var (
	app       *tview.Application
	pages     *tview.Pages
	opsView   *tview.Table
	panes     *tview.Flex
	wrapPanes *tview.Flex
	prevPane  *dirPane

	paneToggle   bool
	layoutToggle bool
)

func newDirPane() *dirPane {
	return &dirPane{
		mode:   initMode,
		path:   initPath,
		apath:  initAPath,
		dpath:  initLPath,
		table:  tview.NewTable(),
		plock:  semaphore.NewWeighted(1),
		hidden: true,
	}
}

func setupUI() {
	app = tview.NewApplication()
	pages = tview.NewPages()

	pages.AddPage("main", setupPaneView(), true, true)
	pages.AddPage("ops", setupOpsView(), true, true)

	pages.SwitchToPage("main")

	if err := app.SetRoot(pages, true).SetFocus(prevPane.table).Run(); err != nil {
		panic(err)
	}
}

func setupPaneView() *tview.Flex {
	selPane, auxPane := newDirPane(), newDirPane()

	prevPane = selPane

	setupPane(selPane, auxPane)
	setupPane(auxPane, selPane)

	panes = tview.NewFlex().
		AddItem(selPane.table, 0, 1, true).
		AddItem(auxPane.table, 0, 1, false).
		SetDirection(tview.FlexColumn)

	wrapPanes = tview.NewFlex().
		AddItem(panes, 0, 2, true).
		SetDirection(tview.FlexRow)

	mainflex := tview.NewFlex().
		AddItem(wrapPanes, 0, 2, false)

	appflex := tview.NewFlex().
		AddItem(mainflex, 0, 1, true)

	appflex.SetBorder(true)
	appflex.SetTitle("| ADBTuiFM |")
	appflex.SetBackgroundColor(tcell.Color16)

	selPane.table.SetBackgroundColor(tcell.Color16)
	auxPane.table.SetBackgroundColor(tcell.Color16)

	return appflex
}

func setupOpsView() *tview.Table {
	if opsView != nil {
		opsView.Clear()
		goto Header
	}

	opsView = tview.NewTable()

	opsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			setResume(false)
			pages.SwitchToPage("main")
			app.SetFocus(prevPane.table)
			opsView.SetSelectable(false, false)
		}

		switch event.Rune() {
		case 'o':
			setResume(false)
			pages.SwitchToPage("main")
			app.SetFocus(prevPane.table)
			opsView.SetSelectable(false, false)

		case 'x':
			r, _ := opsView.GetSelection()
			id, _ := strconv.Atoi(opsView.GetCell(r, 0).Text)
			cancelOps(id)

		case 'C':
			clearAllOps()

		case 'X':
			go cancelAllOps()

		case 'q':
			stopApp()
		}

		return event
	})

Header:
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
	selPane.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		prevPane = selPane

		switch event.Key() {
		case tcell.KeyEscape:
			reset(selPane, auxPane)

		case tcell.KeyTab:
			paneswitch(selPane, auxPane)

		case tcell.KeyEnter, tcell.KeyRight:
			selPane.ChangeDir(true, false)
			return nil

		case tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyLeft:
			selPane.ChangeDir(false, true)
			return nil
		}

		switch event.Rune() {
		case 'o':
			opsPage()

		case 'q':
			stopApp()

		case '?':
			showHelpModal()

		case 'h':
			selPane.setHidden()

		case '/':
			selPane.showFilterInput()

		case 's':
			selPane.modeSwitchHandler()

		case 'g':
			selPane.showChangeDirInput()
			return nil

		case 'r':
			selPane.ChangeDir(false, false)

		case 'S':
			showEditSelections()

		case '[':
			swapLayout()

		case ']':
			swapPanes(selPane, auxPane)

		case 'A', 'a', ',':
			multiselect(selPane, event.Rune())

		case 'm', 'p', 'd':
			opsHandler(selPane, auxPane, event.Rune())

		case 'M', 'R':
			showMkdirRenameInput(selPane, auxPane, event.Rune())
		}

		return event
	})

	selPane.table.SetBorder(true)
	selPane.table.SetSelectable(true, true)

	selPane.ChangeDir(false, false)
}

func opsPage() {
	setResume(true)
	app.SetFocus(opsView)
	pages.SwitchToPage("ops")
	opsView.SetSelectable(true, false)
}

func paneswitch(selPane, auxPane *dirPane) {
	auxPane.reselect(false)
	app.SetFocus(auxPane.table)
	selPane.table.SetSelectable(false, false)
	auxPane.table.SetSelectable(true, false)
}

func reset(selPane, auxPane *dirPane) {
	selected = false
	multiselection = make(map[string]ifaceMode)

	selPane.table.SetSelectable(false, false)

	selPane.reselect(true)
	auxPane.reselect(true)

	app.SetFocus(selPane.table)
	selPane.table.SetSelectable(true, false)
	auxPane.table.SetSelectable(false, false)
}

func multiselect(selPane *dirPane, key rune) {
	var all, inverse bool

	switch key {
	case 'A':
		all = true
		inverse = false

	case 'a':
		all = false
		inverse = true

	case ',':
		all = false
		inverse = false
	}

	totalrows := selPane.table.GetRowCount()

	if totalrows <= 0 {
		return
	}

	selPane.multiSelectHandler(all, inverse, totalrows)
}

func (p *dirPane) reselect(reset bool) {
	if !p.getLock() {
		return
	}
	defer p.setUnlock()

	for i := 0; i < p.table.GetRowCount(); i++ {
		cell := p.table.GetCell(i, 0)

		if reset {
			p.updateDirPane(i, false, cell)
			continue
		}

		if checkSelected(p.path, cell.Text, false) {
			p.updateDirPane(i, true, cell)
		} else {
			p.updateDirPane(i, false, cell)
		}
	}
}

func swapLayout() {
	if !layoutToggle {
		layoutToggle = true
		panes.SetDirection(tview.FlexRow)
		wrapPanes.SetDirection(tview.FlexColumn)
	} else {
		layoutToggle = false
		panes.SetDirection(tview.FlexColumn)
		wrapPanes.SetDirection(tview.FlexRow)
	}
}

func swapPanes(selPane, auxPane *dirPane) {
	if !paneToggle {
		paneToggle = true
		panes.RemoveItem(selPane.table)
		panes.AddItem(selPane.table, 0, 1, true)
	} else {
		paneToggle = false
		panes.RemoveItem(auxPane.table)
		panes.AddItem(auxPane.table, 0, 1, true)
	}
}

func (p *dirPane) updateDirPane(row int, sel bool, cell *tview.TableCell, name ...string) {
	var tablecell *tview.TableCell

	color := tcell.ColorSkyblue

	if sel {
		color = tcell.ColorOrange
	}

	if cell != nil {
		tablecell = cell
	} else {
		tablecell = tview.NewTableCell(name[0])
	}

	p.table.SetCell(row, 0, tablecell.SetTextColor(color))
}

func (p *dirPane) updateRow(lock bool) {
	if !lock {
		p.row, _ = p.table.GetSelection()
		return
	}

	app.QueueUpdateDraw(func() {
		p.row, _ = p.table.GetSelection()
	})
}

func (p *dirPane) setPaneTitle() {
	prefix := ""

	switch p.mode {
	case mAdb:
		prefix = "Adb"

	case mLocal:
		prefix = "Local"
	}

	switch {
	case p.path == "./" || p.path == "../":
		p.path = "/"

	default:
		p.path = trimPath(p.path, false)
	}

	title := fmt.Sprintf("|- %s: %s -|", prefix, p.path)
	p.table.SetTitle(title)
}

func (p *dirPane) setPaneSelectable(status bool) {
	if status {
		prevPane.table.SetSelectable(true, false)
		return
	}

	app.QueueUpdateDraw(func() {
		p.table.SetSelectable(false, false)
	})
}

func (p *dirPane) setHidden() {
	if !p.getLock() {
		return
	}
	defer p.setUnlock()

	if p.hidden {
		p.hidden = false
	} else {
		p.hidden = true
	}

	p.ChangeDir(false, false)
}

func (p *dirPane) getHidden() bool {
	return p.hidden
}

func (p *dirPane) setUnlock() {
	p.plock.Release(1)
}

func (p *dirPane) getLock() bool {
	return p.plock.TryAcquire(1)
}

func stopApp() {
	showConfirmModal("Do you want to quit?", false, func() {
		app.Stop()
		cancelAllOps()
	}, func() {})
}
