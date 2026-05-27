//go:build cgo

package fyneapp

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// segmentedControl — pill track for ≤4 options; same API as RadioGroup.Selected / OnChanged.
type segmentedControl struct {
	track   fyne.CanvasObject
	opts    []string
	btns    []*widget.Button
	selected string
	OnChanged func(string)
}

func newSegmentedControl(options []string) *segmentedControl {
	s := &segmentedControl{opts: append([]string(nil), options...)}
	var row []fyne.CanvasObject
	for _, opt := range options {
		opt := opt
		b := widget.NewButton(opt, func() { s.selectOption(opt) })
		b.Importance = widget.LowImportance
		s.btns = append(s.btns, b)
		row = append(row, b)
	}
	bg := canvas.NewRectangle(colorSurface2)
	bg.CornerRadius = radiusMd
	inner := container.NewHBox(row...)
	s.track = container.NewStack(bg, container.NewPadded(inner))
	return s
}

func (s *segmentedControl) Object() fyne.CanvasObject { return s.track }

func (s *segmentedControl) SetSelected(v string) {
	s.selected = v
	s.refreshStyles()
}

func (s *segmentedControl) Selected() string { return s.selected }

func (s *segmentedControl) selectOption(opt string) {
	if s.selected == opt {
		return
	}
	s.selected = opt
	s.refreshStyles()
	if s.OnChanged != nil {
		s.OnChanged(opt)
	}
}

func (s *segmentedControl) refreshStyles() {
	for i, b := range s.btns {
		if i < len(s.opts) && s.opts[i] == s.selected {
			b.Importance = widget.HighImportance
		} else {
			b.Importance = widget.LowImportance
		}
	}
}

// segmentedList — full-width rows for route presets (vertical).
type segmentedList struct {
	rows     []fyne.CanvasObject
	opts     []string
	selected string
	OnChanged func(string)
}

func newSegmentedList(options []string) *segmentedList {
	sl := &segmentedList{opts: append([]string(nil), options...)}
	for _, opt := range options {
		opt := opt
		row := sl.makeRow(opt)
		sl.rows = append(sl.rows, row)
	}
	return sl
}

func (sl *segmentedList) makeRow(opt string) fyne.CanvasObject {
	btn := widget.NewButton(opt, func() { sl.selectOption(opt) })
	btn.Importance = widget.LowImportance
	return btn
}

func (sl *segmentedList) Object() fyne.CanvasObject {
	return container.NewVBox(sl.rows...)
}

func (sl *segmentedList) SetSelected(v string) {
	sl.selected = v
	sl.refreshStyles()
}

func (sl *segmentedList) Selected() string { return sl.selected }

func (sl *segmentedList) selectOption(opt string) {
	if sl.selected == opt {
		return
	}
	sl.selected = opt
	sl.refreshStyles()
	if sl.OnChanged != nil {
		sl.OnChanged(opt)
	}
}

func (sl *segmentedList) refreshStyles() {
	for i, row := range sl.rows {
		if i >= len(sl.opts) {
			continue
		}
		if btn, ok := row.(*widget.Button); ok {
			if sl.opts[i] == sl.selected {
				btn.Importance = widget.HighImportance
			} else {
				btn.Importance = widget.LowImportance
			}
		}
	}
}

func primaryButton(label string, fn func()) *widget.Button {
	b := widget.NewButton(label, fn)
	b.Importance = widget.HighImportance
	return b
}

func secondaryButton(label string, fn func()) *widget.Button {
	b := widget.NewButton(label, fn)
	b.Importance = widget.LowImportance
	return b
}

func statRow(label string, value *widget.Label) fyne.CanvasObject {
	lbl := captionLabel(label)
	if value == nil {
		value = widget.NewLabel("—")
	}
	styleStat(value)
	return container.NewBorder(nil, nil, lbl, value, nil)
}

func captionLabel(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.Importance = widget.LowImportance
	return l
}

func bodyLabel(text string) *widget.Label {
	return widget.NewLabel(text)
}

func displayLabel(text string) *widget.Label {
	return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

func styleStat(l *widget.Label) {
	l.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	l.Alignment = fyne.TextAlignTrailing
}

type profileListRow struct {
	name  *widget.Label
	typ   *widget.Label
	delay *widget.Label
	obj   fyne.CanvasObject
}

func newProfileListRow() *profileListRow {
	name := widget.NewLabel("")
	name.TextStyle = fyne.TextStyle{Bold: true}
	name.Truncation = fyne.TextTruncateEllipsis
	typ := captionLabel("")
	delay := widget.NewLabel("")
	delay.TextStyle = fyne.TextStyle{Monospace: true}
	delayBadge := delayBadgeWrap(delay)
	top := container.NewBorder(nil, nil, nil, delayBadge, name)
	return &profileListRow{
		name:  name,
		typ:   typ,
		delay: delay,
		obj:   container.NewVBox(top, typ),
	}
}

func profileListRowTemplate() fyne.CanvasObject {
	return newProfileListRow().obj
}

func fillProfileListRow(o fyne.CanvasObject, name, typ, delay string) {
	vbox := o.(*fyne.Container)
	row := vbox.Objects[0].(*fyne.Container)
	nameLbl := row.Objects[0].(*widget.Label)
	badgeStack := row.Objects[1].(*fyne.Container)
	delayLbl := badgeStack.Objects[1].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*widget.Label)
	typLbl := vbox.Objects[1].(*widget.Label)
	nameLbl.SetText(name)
	typLbl.SetText(typ)
	delayLbl.SetText(delay)
}

func delayBadgeWrap(delayLbl *widget.Label) fyne.CanvasObject {
	bg := canvas.NewRectangle(colorSurface2)
	bg.CornerRadius = radiusSm
	return container.NewStack(bg, container.NewPadded(container.NewCenter(delayLbl)))
}

func configToolbar(primary, secondary, danger []fyne.CanvasObject) fyne.CanvasObject {
	var left, mid, right []fyne.CanvasObject
	left = append(left, primary...)
	mid = append(mid, secondary...)
	right = append(right, danger...)
	return container.NewBorder(nil, nil,
		container.NewHBox(left...),
		container.NewHBox(right...),
		container.NewHBox(mid...),
	)
}

func heroConnectCard(inner fyne.CanvasObject) fyne.CanvasObject {
	return accentCard(inner)
}

func settingsSection(title, subtitle string, body fyne.CanvasObject) fyne.CanvasObject {
	head := sectionHeader(title, subtitle)
	return surfaceCard(container.NewVBox(head, vSpace(space2), body), cardPadding())
}

func cardPadding() float32 { return space4 }
