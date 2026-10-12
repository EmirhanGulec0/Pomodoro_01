package main

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Host: platform kabuğunun sağladığı hizmetler (Windows'ta Win32, testte sahte).
type Host interface {
	Measure(f Font, s string) int
	DrawText(cmds []TextCmd)
	PlaySound(wav []byte)
	Notify(title, body string)
	Attention()
	SetTopmost(on bool)
	SetCompact(on bool)
	SystemDark() bool
	ReduceMotion() bool
	SetTitle(s string)
	SetProgress(state int, frac float64) // 0 yok, 1 çalışıyor, 2 duraklatıldı
	SetTrayTip(s string)
	SetChrome(dark bool, caption, text RGB)
	Clipboard() string
}

const (
	faceText = 0
	faceDisp = 1
)

type Font struct {
	Face    uint8
	Size    int
	Weight  int
	Spacing int
}

const (
	alignL = 0
	alignC = 1
	alignR = 2
)

type TextCmd struct {
	S     string
	X, Y  int
	Align int8
	F     Font
	C     RGB
	Clip  bool
	CX0   int
	CX1   int
}

const (
	curArrow = 0
	curHand  = 1
	curText  = 2
)

type Hit struct {
	X0, Y0, X1, Y1 float64
	ID             int
	Tip, Key       string
	Cur            int
	Below          bool
}

// Tıklanabilir öğe kimlikleri.
const (
	idNone = iota
	idScrim
	idPanel
	idSeg0
	idSeg1
	idSeg2
	idStats
	idSettings
	idTheme
	idCompact
	idPlay
	idReset
	idSkip
	idTask
	idRing
	idMinus
	idPlus
	idModalClose
	idHelpLink
	idExpand
	idGoal
	idStepBase   = 100 // + satır*2 + (0 azalt / 1 artır)
	idToggleBase = 200 // + anahtar
	idSegBase    = 300 // + grup*10 + seçenek
	idSlider     = 400
	idChartBar   = 500 // + gün
)

const (
	modalNone = iota
	modalSettings
	modalStats
	modalHelp
)

type App struct {
	W, H int
	u    float64
	c    *Canvas
	host Host
	S    *Settings
	path string
	M    Model

	now   time.Time
	lastT time.Time
	t     float64

	sysDark bool
	pal     Pal
	palFrom Pal
	palTo   Pal
	palT    float64
	palKey  [2]int

	bg    *Canvas
	bgKey bgKey

	mx, my  float64
	mouseIn bool
	pressID int
	dragID  int
	hits    []Hit
	hoverID int
	hoverT0 float64
	hov     map[int]float64
	tgl     map[int]float64
	texts   []TextCmd

	modal     int
	modalOpen bool
	modalT    float64
	mu        float64 // modal içi ölçek
	mA        float64 // modal içi piksel alfa çarpanı
	fadeT     float64
	fadeBg    RGB
	fadeOn    bool

	segX    float64
	ringHov float64

	editing bool
	edit    []rune
	caret   int
	caretT0 float64

	toast   string
	toastT0 float64

	intro float64
	pulse float64

	compact bool

	dirty      bool
	animating  bool
	lastRender float64
	lastSecs   int
	lastBlink  int

	saveAt float64

	sound    []byte
	soundKey [2]int

	lastTitle, lastTip string
	lastProg           [2]int
	chromeKey          [2]int

	chartHover int
	sliderX0   float64
	sliderX1   float64
	reduce     bool
	tipBelow   bool

	taskTX    float64
	taskShift float64
	taskFont  Font
}

func NewApp(host Host, path string) *App {
	s := LoadSettings(path)
	a := &App{
		host: host, S: s, path: path,
		hov: map[int]float64{}, tgl: map[int]float64{},
		u: 1, chartHover: -1, fadeT: 1, lastSecs: -1, mA: 1,
		chromeKey: [2]int{-1, -1},
	}
	a.M = NewModel(s.Durations(), s.LongEvery)
	a.sysDark = host.SystemDark()
	a.reduce = host.ReduceMotion()
	a.palKey = [2]int{b2i(a.dark()), int(a.M.Mode)}
	a.pal = palette(a.dark(), a.M.Mode)
	a.palTo, a.palFrom, a.palT = a.pal, a.pal, 1
	a.lastT = time.Now()
	a.now = a.lastT
	a.dirty = true
	for k, on := range []bool{s.AutoBreaks, s.AutoFocus, s.Notify, s.Topmost} {
		a.tgl[idToggleBase+k] = float64(b2i(on))
	}
	if a.reduce {
		a.intro = 1
	}
	return a
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (a *App) dark() bool {
	switch a.S.Theme {
	case 1:
		return false
	case 2:
		return true
	}
	return a.sysDark
}

func (a *App) Resize(w, h int, pix []uint32, u float64) {
	a.W, a.H = w, h
	a.c = NewCanvasOn(w, h, pix)
	a.u = math.Max(0.75, u)
	a.bg = nil
	a.dirty = true
}

func (a *App) SystemThemeChanged(dark bool) {
	a.sysDark = dark
	a.dirty = true
}

func (a *App) Compact() bool { return a.compact }

func (a *App) requestSave() { a.saveAt = a.t + 0.8 }

func (a *App) SaveNow() {
	a.saveAt = 0
	_ = a.S.Save(a.path)
}

// ---------------------------------------------------------------------------
// Güncelleme döngüsü
// ---------------------------------------------------------------------------

func approach(v *float64, target, dt, rate float64) bool {
	d := target - *v
	if math.Abs(d) < 0.002 {
		*v = target
		return false
	}
	*v += d * (1 - math.Exp(-dt*rate))
	return true
}

func (a *App) Update(now time.Time) {
	dt := now.Sub(a.lastT).Seconds()
	if dt < 0 {
		dt = 0
	}
	if dt > 0.1 {
		dt = 0.1
	}
	a.lastT, a.now = now, now
	a.t += dt
	rate := 1.0
	if a.reduce {
		rate = 1000
	}
	anim := false

	if a.M.Tick(now) {
		a.complete()
	}

	key := [2]int{b2i(a.dark()), int(a.M.Mode)}
	if key != a.palKey {
		a.palKey = key
		a.palFrom, a.palTo, a.palT = a.pal, palette(key[0] == 1, a.M.Mode), 0
	}
	if a.palT < 1 {
		a.palT = math.Min(1, a.palT+dt*rate/0.38)
		a.pal = a.palFrom.Mix(a.palTo, easeInOut(a.palT))
		anim = true
	}
	if ck := [2]int{key[0], b2i(a.compact)}; ck != a.chromeKey {
		a.chromeKey = ck
		a.host.SetChrome(key[0] == 1, a.palTo.Bg0, a.palTo.Text)
	}

	mt := 0.0
	if a.modalOpen {
		mt = 1
	}
	if a.modal != modalNone {
		if approach(&a.modalT, mt, dt, 16*rate) {
			anim = true
		}
		if !a.modalOpen && a.modalT == 0 {
			a.modal = modalNone
			a.dirty = true
		}
	}
	if approach(&a.segX, float64(a.M.Mode), dt, 14*rate) {
		anim = true
	}
	ringTarget := 0.0
	if (a.hoverID == idRing || a.hoverID == idMinus || a.hoverID == idPlus) && a.adjustable() {
		ringTarget = 1
	}
	if approach(&a.ringHov, ringTarget, dt, 14*rate) {
		anim = true
	}
	for id, v := range a.hov {
		target := 0.0
		if id == a.hoverID {
			target = 1
		}
		if approach(&v, target, dt, 18*rate) {
			anim = true
		}
		if v == 0 && target == 0 {
			delete(a.hov, id)
		} else {
			a.hov[id] = v
		}
	}
	if a.hoverID != 0 {
		if _, ok := a.hov[a.hoverID]; !ok {
			a.hov[a.hoverID] = 0
			anim = true
		}
	}
	for k, on := range []bool{a.S.AutoBreaks, a.S.AutoFocus, a.S.Notify, a.S.Topmost} {
		v := a.tgl[idToggleBase+k]
		if approach(&v, float64(b2i(on)), dt, 16*rate) {
			anim = true
		}
		a.tgl[idToggleBase+k] = v
	}
	if a.intro < 1 {
		a.intro = math.Min(1, a.intro+dt/0.9)
		anim = true
	}
	if a.pulse > 0.001 {
		a.pulse *= math.Exp(-dt * 2.2 * rate)
		anim = true
	} else {
		a.pulse = 0
	}
	if a.toast != "" {
		if a.t-a.toastT0 > 2.9 {
			a.toast = ""
			a.dirty = true
		} else {
			anim = true
		}
	}
	if a.tipAlpha() > 0 && a.tipAlpha() < 1 {
		anim = true
	}
	if a.hoverID != 0 && a.t-a.hoverT0 > 0.5 && a.t-a.hoverT0 < 0.75 && a.hitTip(a.hoverID) != "" {
		anim = true
	}
	if a.saveAt > 0 && a.t >= a.saveAt {
		a.SaveNow()
	}
	a.syncHost()
	a.animating = anim
}

func (a *App) WantsFrame() bool {
	if a.W == 0 || a.c == nil {
		return false
	}
	if a.dirty || a.animating {
		return true
	}
	if a.M.Secs() != a.lastSecs {
		return true
	}
	if a.M.Running && a.t-a.lastRender >= 0.25 {
		return true
	}
	if a.editing && int((a.t-a.caretT0)/0.53) != a.lastBlink {
		return true
	}
	return false
}

func (a *App) syncHost() {
	title := "Pomodoro"
	tip := "Pomodoro"
	if a.M.Running || a.M.Started() {
		state := "kaldı"
		if !a.M.Running {
			state = "duraklatıldı"
		}
		title = fmt.Sprintf("%s %s – Pomodoro", a.M.Clock(), modeName[a.M.Mode])
		tip = fmt.Sprintf("Pomodoro\n%s: %s %s", modeName[a.M.Mode], a.M.Clock(), state)
	}
	if title != a.lastTitle {
		a.lastTitle = title
		a.host.SetTitle(title)
	}
	if tip != a.lastTip {
		a.lastTip = tip
		a.host.SetTrayTip(tip)
	}
	st := 0
	if a.M.Running {
		st = 1
	} else if a.M.Started() {
		st = 2
	}
	prog := [2]int{st, int((1 - a.M.Frac()) * 1000)}
	if prog != a.lastProg {
		a.lastProg = prog
		a.host.SetProgress(st, 1-a.M.Frac())
	}
}

// ---------------------------------------------------------------------------
// Oturum mantığı
// ---------------------------------------------------------------------------

func (a *App) record(full bool) {
	if a.M.Mode != Focus {
		return
	}
	el := a.M.Elapsed()
	if full {
		el = a.M.Dur[Focus]
	}
	if !full && el < time.Minute {
		return
	}
	a.S.AddRecord(Record{End: a.now.Unix(), Sec: int(el.Seconds()), Task: a.S.Task, Full: full})
	a.requestSave()
}

func (a *App) complete() {
	finished := a.M.Mode
	if finished == Focus {
		a.record(true)
	}
	a.M.Advance(finished == Focus)
	next := a.M.Mode
	a.pulse = 1
	var title, body string
	mins := int(a.M.Dur[next] / time.Minute)
	switch {
	case finished == Focus && next == Long:
		title, body = "Odak tamamlandı", fmt.Sprintf("Uzun mola zamanı: %d dk. Bu seti bitirdin.", mins)
	case finished == Focus:
		title, body = "Odak tamamlandı", fmt.Sprintf("Kısa mola zamanı: %d dk.", mins)
	default:
		title, body = "Mola bitti", fmt.Sprintf("Sıradaki odak: %d dk.", mins)
		if a.S.Task != "" {
			body = fmt.Sprintf("Sıradaki odak: %d dk, %s.", mins, a.S.Task)
		}
	}
	if a.S.Notify {
		a.host.Notify(title, body)
	}
	a.host.Attention()
	a.playSound()
	a.showToast(title + ". " + body)
	if (next != Focus && a.S.AutoBreaks) || (next == Focus && a.S.AutoFocus) {
		a.M.Start(a.now)
	}
	a.dirty = true
}

func (a *App) playSound() {
	if a.S.Sound >= 3 || a.S.Volume == 0 {
		return
	}
	k := [2]int{a.S.Sound, a.S.Volume}
	if a.sound == nil || k != a.soundKey {
		a.sound, a.soundKey = MakeWAV(a.S.Sound, a.S.Volume), k
	}
	a.host.PlaySound(a.sound)
}

func (a *App) toggle() {
	if a.M.Running {
		a.M.Pause(a.now)
	} else {
		a.M.Start(a.now)
		if a.intro < 1 {
			a.intro = 1
		}
	}
	a.dirty = true
}

func (a *App) reset() {
	a.record(false)
	a.M.Reset()
	a.dirty = true
}

func (a *App) skip() {
	a.record(false)
	a.M.Advance(false)
	a.pulse = 0.5
	a.dirty = true
}

func (a *App) setMode(md Mode) {
	if md == a.M.Mode && !a.M.Started() {
		return
	}
	a.record(false)
	a.M.SetMode(md)
	a.dirty = true
}

func (a *App) adjustable() bool { return !a.M.Running && !a.M.Started() }

func (a *App) adjust(delta int) {
	if !a.adjustable() {
		if a.M.Running || a.M.Started() {
			a.showToast("Süreyi değiştirmek için önce sıfırla")
		}
		return
	}
	switch a.M.Mode {
	case Focus:
		a.S.FocusMin = clampi(a.S.FocusMin+delta, 1, 180)
	case Short:
		a.S.ShortMin = clampi(a.S.ShortMin+delta, 1, 60)
	case Long:
		a.S.LongMin = clampi(a.S.LongMin+delta, 1, 90)
	}
	a.applyDurations()
}

func (a *App) applyDurations() {
	a.M.SetDurations(a.S.Durations(), a.S.LongEvery)
	a.requestSave()
	a.dirty = true
}

func (a *App) showToast(s string) {
	a.toast = s
	a.toastT0 = a.t
	a.dirty = true
}

func (a *App) toggleTheme() {
	if a.dark() {
		a.S.Theme = 1
		a.showToast("Açık tema")
	} else {
		a.S.Theme = 2
		a.showToast("Koyu tema")
	}
	a.requestSave()
}

func (a *App) toggleMute() {
	if a.S.Sound == 3 {
		a.S.Sound = a.S.LastSound
		a.showToast("Ses açık: " + soundName[a.S.Sound])
		a.playSound()
	} else {
		a.S.LastSound = a.S.Sound
		a.S.Sound = 3
		a.showToast("Ses kapalı")
	}
	a.requestSave()
}

func (a *App) setTopmost(on bool) {
	a.S.Topmost = on
	if !a.compact {
		a.host.SetTopmost(on)
	}
	if on {
		a.showToast("Pencere her zaman üstte")
	} else {
		a.showToast("Pencere artık üstte kalmıyor")
	}
	a.requestSave()
}

func (a *App) SetCompactMode(on bool) {
	if on == a.compact {
		return
	}
	a.commitEdit()
	a.modal, a.modalOpen, a.modalT = modalNone, false, 0
	a.compact = on
	a.S.StartMini = on
	a.hoverID, a.pressID = 0, 0
	a.requestSave()
	a.host.SetCompact(on)
	a.dirty = true
}

func (a *App) openModal(m int) {
	if a.modal == m && a.modalOpen {
		a.closeModal()
		return
	}
	a.commitEdit()
	if a.modal == modalNone {
		a.modalT = 0
	}
	a.modal, a.modalOpen = m, true
	a.chartHover = -1
	a.dirty = true
}

func (a *App) closeModal() {
	a.modalOpen = false
	a.dragID = 0
	a.dirty = true
}

// ---------------------------------------------------------------------------
// Görev düzenleme
// ---------------------------------------------------------------------------

func (a *App) beginEdit() {
	if a.compact {
		return
	}
	a.editing = true
	a.edit = []rune(a.S.Task)
	a.caret = len(a.edit)
	a.caretT0 = a.t
	a.dirty = true
}

func (a *App) commitEdit() {
	if !a.editing {
		return
	}
	a.editing = false
	task := strings.TrimSpace(string(a.edit))
	if task != a.S.Task {
		a.S.Task = task
		a.requestSave()
	}
	a.dirty = true
}

func (a *App) cancelEdit() {
	a.editing = false
	a.dirty = true
}

func (a *App) insertText(s string) {
	for _, r := range s {
		if r < 32 || r == 127 || len(a.edit) >= 80 {
			continue
		}
		a.edit = append(a.edit, 0)
		copy(a.edit[a.caret+1:], a.edit[a.caret:])
		a.edit[a.caret] = r
		a.caret++
	}
	a.caretT0 = a.t
	a.dirty = true
}

func (a *App) editKey(vk int, ctrl bool) {
	switch vk {
	case 0x0D:
		a.commitEdit()
	case 0x1B:
		a.cancelEdit()
	case 0x25:
		if a.caret > 0 {
			a.caret--
		}
	case 0x27:
		if a.caret < len(a.edit) {
			a.caret++
		}
	case 0x24:
		a.caret = 0
	case 0x23:
		a.caret = len(a.edit)
	case 0x2E:
		if a.caret < len(a.edit) {
			a.edit = append(a.edit[:a.caret], a.edit[a.caret+1:]...)
		}
	case 'V':
		if ctrl {
			a.insertText(strings.ReplaceAll(a.host.Clipboard(), "\n", " "))
		}
	}
	a.caretT0 = a.t
	a.dirty = true
}

// ---------------------------------------------------------------------------
// Girdi
// ---------------------------------------------------------------------------

func (a *App) KeyDown(vk int, ctrl, shift bool) {
	if a.editing {
		a.editKey(vk, ctrl)
		return
	}
	if a.compact {
		switch vk {
		case 0x20, 0x0D:
			a.toggle()
		case 0x1B, 'C':
			a.SetCompactMode(false)
		case 'R':
			a.reset()
		case 'N':
			a.skip()
		}
		return
	}
	if vk == 0x1B {
		if a.modalOpen {
			a.closeModal()
		}
		return
	}
	if a.modalOpen {
		switch vk {
		case 'S':
			a.openModal(modalSettings)
		case 'I':
			a.openModal(modalStats)
		case 0x70, 'H':
			a.openModal(modalHelp)
		}
		return
	}
	switch vk {
	case 0x20, 0x0D:
		a.toggle()
	case 'R':
		a.reset()
	case 'N':
		a.skip()
	case '1', 0x61:
		a.setMode(Focus)
	case '2', 0x62:
		a.setMode(Short)
	case '3', 0x63:
		a.setMode(Long)
	case 0x26, 0x6B, 0xBB:
		a.adjust(1)
	case 0x28, 0x6D, 0xBD:
		a.adjust(-1)
	case 'E':
		a.beginEdit()
	case 'S':
		a.openModal(modalSettings)
	case 'I':
		a.openModal(modalStats)
	case 'D':
		a.toggleTheme()
	case 'C':
		a.SetCompactMode(true)
	case 'P':
		a.setTopmost(!a.S.Topmost)
	case 'M':
		a.toggleMute()
	case 0x70, 'H':
		a.openModal(modalHelp)
	case 0xBC:
		if ctrl {
			a.openModal(modalSettings)
		}
	}
}

func (a *App) Char(r rune) {
	if !a.editing {
		return
	}
	if r == 8 {
		if a.caret > 0 {
			a.edit = append(a.edit[:a.caret-1], a.edit[a.caret:]...)
			a.caret--
		}
		a.caretT0 = a.t
		a.dirty = true
		return
	}
	if r >= 32 && r != 127 {
		a.insertText(string(r))
	}
}

func (a *App) hitAt(x, y float64) *Hit {
	for i := len(a.hits) - 1; i >= 0; i-- {
		h := &a.hits[i]
		if x >= h.X0 && x < h.X1 && y >= h.Y0 && y < h.Y1 {
			return h
		}
	}
	return nil
}

func (a *App) hitTip(id int) string {
	for i := range a.hits {
		if a.hits[i].ID == id {
			return a.hits[i].Tip
		}
	}
	return ""
}

func (a *App) MouseMove(x, y int) {
	a.mx, a.my = float64(x), float64(y)
	a.mouseIn = true
	if a.dragID == idSlider {
		a.setSliderFromX(a.mx)
	}
	id := 0
	if h := a.hitAt(a.mx, a.my); h != nil {
		id = h.ID
	}
	if id == idScrim || id == idPanel {
		id = 0
	}
	if id != a.hoverID {
		a.hoverID = id
		a.hoverT0 = a.t
		a.dirty = true
	}
	ch := -1
	if id >= idChartBar && id < idChartBar+7 {
		ch = id - idChartBar
	}
	if ch != a.chartHover {
		a.chartHover = ch
		a.dirty = true
	}
}

func (a *App) MouseLeave() {
	a.mouseIn = false
	if a.hoverID != 0 || a.chartHover != -1 {
		a.hoverID = 0
		a.chartHover = -1
		a.dirty = true
	}
}

func (a *App) MouseDown(x, y int) {
	a.MouseMove(x, y)
	h := a.hitAt(a.mx, a.my)
	if a.editing && (h == nil || h.ID != idTask) {
		a.commitEdit()
	}
	if h == nil {
		a.pressID = 0
		return
	}
	a.pressID = h.ID
	if h.ID == idSlider && a.S.Sound < 3 {
		a.dragID = idSlider
		a.setSliderFromX(a.mx)
	}
	if h.ID == idTask && a.editing {
		a.placeCaret(a.mx)
	}
	a.dirty = true
}

func (a *App) MouseUp(x, y int) {
	a.mx, a.my = float64(x), float64(y)
	if a.dragID == idSlider {
		a.dragID = 0
		a.pressID = 0
		a.playSound()
		a.dirty = true
		return
	}
	if h := a.hitAt(a.mx, a.my); h != nil && h.ID == a.pressID {
		a.act(h.ID)
	}
	a.pressID = 0
	a.dirty = true
}

func (a *App) Wheel(delta int, x, y int) {
	if a.modalOpen || a.compact {
		return
	}
	h := a.hitAt(float64(x), float64(y))
	if h == nil || (h.ID != idRing && h.ID != idMinus && h.ID != idPlus) {
		return
	}
	if delta > 0 {
		a.adjust(1)
	} else if delta < 0 {
		a.adjust(-1)
	}
}

// Cursor: fare altındaki öğeye göre imleç türü.
func (a *App) Cursor() int {
	if h := a.hitAt(a.mx, a.my); h != nil {
		return h.Cur
	}
	return curArrow
}

// Draggable: kompakt modda pencereyi sürüklenebilir yapan boş alan mı.
func (a *App) Draggable(x, y int) bool {
	if !a.compact {
		return false
	}
	h := a.hitAt(float64(x), float64(y))
	return h == nil
}

func (a *App) setSliderFromX(x float64) {
	if a.sliderX1 <= a.sliderX0 {
		return
	}
	v := int(math.Round(clampf((x-a.sliderX0)/(a.sliderX1-a.sliderX0), 0, 1) * 100))
	if v != a.S.Volume {
		a.S.Volume = v
		a.requestSave()
		a.dirty = true
	}
}

func (a *App) act(id int) {
	switch {
	case id == idPlay:
		a.toggle()
	case id == idReset:
		a.reset()
	case id == idSkip:
		a.skip()
	case id >= idSeg0 && id <= idSeg2:
		a.setMode(Mode(id - idSeg0))
	case id == idStats || id == idGoal:
		a.openModal(modalStats)
	case id == idSettings:
		a.openModal(modalSettings)
	case id == idTheme:
		a.toggleTheme()
	case id == idCompact:
		a.SetCompactMode(true)
	case id == idExpand:
		a.SetCompactMode(false)
	case id == idTask:
		if !a.editing {
			a.beginEdit()
		}
	case id == idMinus:
		a.adjust(-1)
	case id == idPlus:
		a.adjust(1)
	case id == idScrim || id == idModalClose:
		a.closeModal()
	case id == idHelpLink:
		a.openModal(modalHelp)
	case id >= idStepBase && id < idStepBase+20:
		row, dir := (id-idStepBase)/2, (id-idStepBase)%2*2-1
		a.step(row, dir)
	case id >= idToggleBase && id < idToggleBase+10:
		a.flip(id - idToggleBase)
	case id >= idSegBase && id < idSegBase+40:
		a.pick((id-idSegBase)/10, (id-idSegBase)%10)
	}
}

// Ayar satırları: 0 odak, 1 kısa mola, 2 uzun mola, 3 uzun mola sıklığı, 4 günlük hedef.
func (a *App) step(row, dir int) {
	s := a.S
	switch row {
	case 0:
		if s.FocusMin < 5 && dir > 0 {
			s.FocusMin = 5
		} else {
			s.FocusMin = clampi(s.FocusMin+5*dir, 5, 120)
		}
	case 1:
		s.ShortMin = clampi(s.ShortMin+dir, 1, 30)
	case 2:
		s.LongMin = clampi(s.LongMin+5*dir, 5, 60)
	case 3:
		s.LongEvery = clampi(s.LongEvery+dir, 2, 8)
		if a.M.Cycle > s.LongEvery {
			a.M.Cycle = s.LongEvery
		}
	case 4:
		s.Goal = clampi(s.Goal+dir, 1, 24)
	}
	a.applyDurations()
}

// Anahtarlar: 0 molaları otomatik başlat, 1 odakları otomatik başlat, 2 bildirim, 3 üstte tut.
func (a *App) flip(k int) {
	s := a.S
	switch k {
	case 0:
		s.AutoBreaks = !s.AutoBreaks
	case 1:
		s.AutoFocus = !s.AutoFocus
	case 2:
		s.Notify = !s.Notify
	case 3:
		a.setTopmost(!s.Topmost)
		return
	}
	a.requestSave()
	a.dirty = true
}

// Segment grupları: 0 tema, 1 arka plan, 2 bildirim sesi.
func (a *App) pick(g, opt int) {
	s := a.S
	switch g {
	case 0:
		s.Theme = clampi(opt, 0, 2)
	case 1:
		s.Background = clampi(opt, 0, 2)
	case 2:
		s.Sound = clampi(opt, 0, 3)
		if s.Sound < 3 {
			s.LastSound = s.Sound
			a.playSound()
		}
	}
	a.requestSave()
	a.dirty = true
}
