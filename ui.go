package main

import (
	"fmt"

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
	entry    *adb.DirEntry
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

func newDirPane(selpane bool) *dirPane {
	var initPath string
	var initMode ifaceMode

	if selpane {
		initMode = initSelMode
		initPath = initSelPath
	} else {
		initMode = initAuxMode
		initPath = initAuxPath
	}

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

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			return nil

		case tcell.KeyCtrlD:
			execCmd("", "Foreground")
		}

		return event
	})

	app.SetBeforeDrawFunc(func(t tcell.Screen) bool {
		width, _ := t.Size()

		resizePopup(width)
		resizeProgress(width)

		return false
	})

	if err := app.SetRoot(pages, true).SetFocus(prevPane.table).Run(); err != nil {
		panic(err)
	}
}

func setupPaneView() *tview.Flex {
	selPane, auxPane := newDirPane(true), newDirPane(false)

	prevPane = selPane

	setupStatus()
	setupPane(selPane, auxPane)
	setupPane(auxPane, selPane)

	panes = tview.NewFlex().
		AddItem(selPane.table, 0, 1, true).
		AddItem(auxPane.table, 0, 1, false).
		SetDirection(tview.FlexColumn)

	wrapPanes = tview.NewFlex().
		AddItem(panes, 0, 2, true).
		SetDirection(tview.FlexRow)

	wrapView := tview.NewFlex().
		AddItem(wrapPanes, 0, 2, false)

	wrapFlex := tview.NewFlex().
		AddItem(wrapView, 0, 1, true)

	wrapStatus := tview.NewFlex().
		AddItem(wrapFlex, 0, 2, true).
		AddItem(statuspgs, 1, 0, true).
		SetDirection(tview.FlexRow)

	wrapFlex.SetBackgroundColor(tcell.ColorDefault)

	selPane.table.SetBackgroundColor(tcell.ColorDefault)
	auxPane.table.SetBackgroundColor(tcell.ColorDefault)

	return wrapStatus
}

func setupOpsView() *tview.Table {
	opsView = tview.NewTable()

	exit := func() {
		if opsView.HasFocus() {
			pages.SwitchToPage("main")
			app.SetFocus(prevPane.table)
			opsView.SetSelectable(false, false)
		}
	}

	canceltask := func() {
		row, _ := opsView.GetSelection()
		ref := opsView.GetCell(row, 0).GetReference()

		if ref != nil {
			ref.(*operation).cancelOps()
		}
	}

	opsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			exit()
		}

		switch event.Rune() {
		case 'x':
			canceltask()

		case 'X':
			opsView.Clear()
			go cancelAllOps()
			fallthrough

		case 'o':
			exit()

		case 'q':
			pages.SwitchToPage("main")
			stopApp()
		}

		return event
	})

	opsView.SetBorder(true)
	opsView.SetSelectable(false, false)

	opsView.SetTitle("[::bu]Operations")
	opsView.SetTitleAlign(tview.AlignLeft)

	opsView.SetBorderColor(tcell.ColorDefault)
	opsView.SetBackgroundColor(tcell.ColorDefault)

	return opsView
}

//gocyclo:ignore
func setupPane(selPane, auxPane *dirPane) {
	selPane.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		prevPane = selPane

		switch event.Key() {
		case tcell.KeyEscape:
			reset(selPane, auxPane)

		case tcell.KeyTab:
			paneswitch(selPane, auxPane)

		case tcell.KeyCtrlO:
			go selPane.openFileHandler()

		case tcell.KeyCtrlR:
			selPane.reselect(false)

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

		case 'h', '.':
			selPane.setHidden()

		case '/':
			selPane.showFilterInput()

		case 's', '<':
			selPane.modeSwitchHandler()

		case 'g', '>':
			selPane.showChangeDirInput()
			return nil

		case 'r':
			selPane.ChangeDir(false, false)

		case 'S':
			showEditSelections(nil)

		case '[':
			swapLayout(selPane, auxPane)

		case ']':
			swapPanes(selPane, auxPane)

		case '!':
			execCommand()

		case 'A', 'a', ' ':
			multiselect(selPane, event.Rune())

		case 'm', 'p', 'P', 'd':
			opsHandler(selPane, auxPane, event.Rune())

		case 'M', 'R':
			showMkdirRenameInput(selPane, auxPane, event.Rune())
		}

		return event
	})

	selPane.table.SetBorder(true)
	selPane.table.SetSelectable(true, false)

	selPane.table.SetSelectionChangedFunc(func(row, col int) {
		rows := selPane.table.GetRowCount()

		if row < 0 || row > rows {
			return
		}

		cell := selPane.table.GetCell(row, col)

		if cell == nil {
			return
		}

		selPane.table.SetSelectedStyle(tcell.Style{}.
			Background(cell.Color).
			Foreground(tcell.Color16).
			Attributes(cell.Attributes))
	})

	selPane.ChangeDir(false, false)
}

func opsPage() {
	rows := opsView.GetRowCount()

	if rows == 0 {
		showInfoMsg("No operations in queue")
		return
	}

	app.SetFocus(opsView)
	pages.SwitchToPage("ops")
	opsView.SetSelectable(true, false)
}

func paneswitch(selPane, auxPane *dirPane) {
	auxPane.reselect(true)
	app.SetFocus(auxPane.table)
	selPane.table.SetSelectable(false, false)
	auxPane.table.SetSelectable(true, false)
}

func reset(selPane, auxPane *dirPane) {
	selected = false
	multiselection = make(map[string]ifaceMode)

	selPane.table.SetSelectable(false, false)
	selPane.reselect(true)

	app.SetFocus(selPane.table)
	selPane.table.SetSelectable(true, false)
	auxPane.table.SetSelectable(false, false)
}

func resetOpsView() {
	count := opsView.GetRowCount()
	row, _ := opsView.GetSelection()

	switch {
	case count/opRowNum == 0:
		pages.SwitchToPage("main")
		app.SetFocus(prevPane.table)
		opsView.SetSelectable(false, false)

	case row-1 == count:
		opsView.Select(row-opRowNum, 0)
	}
}

func swapLayout(selPane, auxPane *dirPane) {
	if !layoutToggle {
		layoutToggle = true
		panes.SetDirection(tview.FlexRow)
		wrapPanes.SetDirection(tview.FlexColumn)
	} else {
		layoutToggle = false
		panes.SetDirection(tview.FlexColumn)
		wrapPanes.SetDirection(tview.FlexRow)
	}

	selPane.reselect(true)
	auxPane.reselect(true)
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

func multiselect(selPane *dirPane, key rune) {
	var all, inverse bool

	switch key {
	case 'A':
		all = true
		inverse = false

	case 'a':
		all = false
		inverse = true

	case ' ':
		all = false
		inverse = false
	}

	totalrows := selPane.table.GetRowCount()

	if totalrows <= 0 {
		return
	}

	selPane.multiSelectHandler(all, inverse, totalrows)
}

func (p *dirPane) reselect(psel bool) {
	if !p.getLock() {
		return
	}
	defer p.setUnlock()

	for row, dir := range p.pathList {
		checksel := checkSelected(p.path, dir.Name, false)
		p.updateDirPane(row, checksel, nil, dir)
	}

	pos, _ := p.table.GetSelection()
	p.table.Select(pos, 0)
}

func (p *dirPane) updateDirPane(row int, sel bool, cells []*tview.TableCell, dir *adb.DirEntry) {
	entry := getListEntry(dir)

	for col, dname := range entry {
		if !layoutToggle && col > 0 {
			dname = ""
		}

		color, attr := setEntryColor(col, sel, entry[1])

		cell := tview.NewTableCell(tview.Escape(dname))
		cell.SetReference(dir)

		if col > 0 {
			if col == 1 {
				cell.SetExpansion(1)
				cell.SetAlign(tview.AlignRight)
			}

			cell.SetSelectable(false)
		}

		p.table.SetCell(row, col, cell.SetTextColor(color).
			SetAttributes(attr))
	}
}

func (p *dirPane) updateRef(lock bool) {
	update := func() {
		p.row, _ = p.table.GetSelection()

		ref := p.table.GetCell(p.row, 0).GetReference()

		if ref != nil {
			p.entry = ref.(*adb.DirEntry)
		} else {
			p.entry = nil
		}
	}

	if !lock {
		update()
		return
	}

	app.QueueUpdateDraw(func() {
		update()
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

	title := fmt.Sprintf("|- %s: %s -|", prefix, tview.Escape(p.path))
	p.table.SetTitle(title)
}

func (p *dirPane) setPaneSelectable(status bool) {
	if status {
		if p.table.HasFocus() {
			p.table.SetSelectable(true, false)
		}

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
		showInfoMsg("Showing hidden files")
	} else {
		p.hidden = true
		showInfoMsg("Hiding hidden files")
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
	quitmsg := "Quit"

	istask := opsView.GetRowCount()
	if istask > 0 {
		quitmsg += " (jobs are still running)"
	}

	quitmsg += " (y/n)?"

	showConfirmMsg(quitmsg, func() {
		stopUI()
	}, func() {})
}

func stopUI() {
	app.Stop()
	stopStatus()
	cancelAllOps()
}
