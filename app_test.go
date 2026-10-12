package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeHost struct {
	notes    []string
	sounds   int
	compact  bool
	topmost  bool
	drawn    int
	progress [2]float64
}

func (h *fakeHost) Measure(f Font, s string) int {
	return int(float64(len([]rune(s))) * float64(f.Size) * 0.55)
}
func (h *fakeHost) DrawText(c []TextCmd)         { h.drawn += len(c) }
func (h *fakeHost) PlaySound(b []byte)           { h.sounds++ }
func (h *fakeHost) Notify(t, b string)           { h.notes = append(h.notes, t+"|"+b) }
func (h *fakeHost) Attention()                   {}
func (h *fakeHost) SetTopmost(on bool)           { h.topmost = on }
func (h *fakeHost) SetCompact(on bool)           { h.compact = on }
func (h *fakeHost) SystemDark() bool             { return true }
func (h *fakeHost) ReduceMotion() bool           { return false }
func (h *fakeHost) SetTitle(string)              {}
func (h *fakeHost) SetProgress(s int, f float64) { h.progress = [2]float64{float64(s), f} }
func (h *fakeHost) SetTrayTip(string)            {}
func (h *fakeHost) SetChrome(bool, RGB, RGB)     {}
func (h *fakeHost) Clipboard() string            { return "panodan\nmetin" }

func newTestApp(t *testing.T, w, h int) (*App, *fakeHost, *time.Time) {
	fh := &fakeHost{}
	path := filepath.Join(t.TempDir(), "ayarlar.json")
	a := NewApp(fh, path)
	a.Resize(w, h, make([]uint32, w*h), 1)
	now := time.Now()
	a.lastT = now
	return a, fh, &now
}

func adv(a *App, now *time.Time, d time.Duration) {
	*now = now.Add(d)
	a.Update(*now)
}

func TestCycleAndRecords(t *testing.T) {
	a, fh, now := newTestApp(t, 1000, 700)
	a.S.FocusMin, a.S.ShortMin, a.S.LongMin, a.S.LongEvery = 1, 1, 1, 4
	a.applyDurations()
	want := []Mode{Short, Focus, Short, Focus, Short, Focus, Long, Focus}
	for i, w := range want {
		a.toggle()
		adv(a, now, 61*time.Second)
		if a.WantsFrame() {
			a.Render()
		}
		if a.M.Mode != w {
			t.Fatalf("adım %d: mod=%v beklenen=%v", i, a.M.Mode, w)
		}
	}
	if got := a.S.Day(*now).Count; got != 4 {
		t.Fatalf("bugün tamamlanan=%d, beklenen 4", got)
	}
	if len(fh.notes) != len(want) || fh.sounds != len(want) {
		t.Fatalf("bildirim=%d ses=%d", len(fh.notes), fh.sounds)
	}
	if a.M.Cycle != 0 {
		t.Fatalf("uzun moladan sonra set sıfırlanmalı: %d", a.M.Cycle)
	}
}

func TestAutoStartAndPartial(t *testing.T) {
	a, _, now := newTestApp(t, 1000, 700)
	a.S.FocusMin, a.S.AutoBreaks = 2, true
	a.applyDurations()
	a.toggle()
	adv(a, now, 2*time.Minute+time.Second)
	if a.M.Mode != Short || !a.M.Running {
		t.Fatalf("mola otomatik başlamalıydı: mod=%v çalışıyor=%v", a.M.Mode, a.M.Running)
	}
	// yarım odak kaydı
	a.setMode(Focus)
	a.toggle()
	adv(a, now, 90*time.Second)
	a.skip()
	recs := a.S.TodayRecords(*now)
	if len(recs) != 2 || recs[0].Full || recs[0].Sec < 89 {
		t.Fatalf("yarım kayıt bekleniyordu: %+v", recs)
	}
	// 1 dakikadan kısa yarım oturum kaydedilmez
	a.setMode(Focus)
	a.toggle()
	adv(a, now, 20*time.Second)
	a.reset()
	if n := len(a.S.TodayRecords(*now)); n != 2 {
		t.Fatalf("kısa oturum kaydedilmemeli, kayıt=%d", n)
	}
}

func TestStatsStreak(t *testing.T) {
	s := DefaultSettings()
	now := time.Date(2026, 10, 12, 15, 0, 0, 0, time.Local)
	add := func(daysAgo int, full bool) {
		s.Records = append(s.Records, Record{End: now.AddDate(0, 0, -daysAgo).Add(-time.Hour).Unix(), Sec: 1500, Full: full})
	}
	add(1, true)
	add(2, true)
	add(3, false)
	add(4, true)
	if got := s.Streak(now); got != 2 {
		t.Fatalf("seri=%d, beklenen 2 (bugün yok, dün+önceki gün)", got)
	}
	add(0, true)
	if got := s.Streak(now); got != 3 {
		t.Fatalf("seri=%d, beklenen 3", got)
	}
	days := s.LastDays(now, 7)
	if len(days) != 7 || !days[6].Date.Equal(dayStart(now)) || days[6].Count != 1 || days[3].Sec != 1500 || days[3].Count != 0 {
		t.Fatalf("son 7 gün hatalı: %+v", days)
	}
	if c, sec := s.Totals(); c != 4 || sec != 7500 {
		t.Fatalf("toplam %d %d", c, sec)
	}
	if fmtDur(4500) != "1 sa 15 dk" || fmtDur(3600) != "1 sa" || fmtDur(59) != "0 dk" {
		t.Fatal("fmtDur")
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x", "ayarlar.json")
	s := DefaultSettings()
	s.FocusMin, s.Theme, s.Task = 50, 2, "Görev ğüşiöç"
	s.Records = []Record{{End: 1, Sec: 60, Full: true}}
	if err := s.Save(p); err != nil {
		t.Fatal(err)
	}
	r := LoadSettings(p)
	if r.FocusMin != 50 || r.Theme != 2 || r.Task != "Görev ğüşiöç" || len(r.Records) != 1 {
		t.Fatalf("geri okuma hatalı: %+v", r)
	}
	os.WriteFile(p, []byte("{bozuk"), 0o644)
	r = LoadSettings(p)
	if r.FocusMin != 25 {
		t.Fatal("bozuk dosyada varsayılana dönülmeli")
	}
	if _, err := os.Stat(p + ".bozuk"); err != nil {
		t.Fatal("bozuk dosya yedeklenmeli")
	}
}

func TestWAV(t *testing.T) {
	for k := 0; k < 3; k++ {
		b := MakeWAV(k, 70)
		if !bytes.HasPrefix(b, []byte("RIFF")) || string(b[8:12]) != "WAVE" || len(b) < 44+44100 {
			t.Fatalf("WAV %d hatalı", k)
		}
	}
}

func TestEditing(t *testing.T) {
	a, _, now := newTestApp(t, 1000, 700)
	a.Render()
	a.KeyDown('E', false, false)
	for _, r := range "Ödev" {
		a.Char(r)
	}
	a.Char(8)
	a.KeyDown(0x25, false, false)
	a.Char('x')
	a.KeyDown('V', true, false)
	a.KeyDown(0x0D, false, false)
	if a.S.Task != "Ödxpanodan metine" {
		t.Fatalf("görev=%q", a.S.Task)
	}
	// düzenleme sırasında kısayollar çalışmaz
	a.KeyDown('E', false, false)
	a.KeyDown(0x20, false, false)
	if a.M.Running {
		t.Fatal("düzenlerken boşluk zamanlayıcıyı başlatmamalı")
	}
	a.KeyDown(0x1B, false, false)
	adv(a, now, time.Second)
	if a.editing {
		t.Fatal("Esc düzenlemeyi kapatmalı")
	}
}

// Tüm ekranlar farklı boyut ve ölçeklerde çökmeden çizilmeli.
func TestRenderAll(t *testing.T) {
	sizes := []struct {
		w, h int
		u    float64
	}{{760, 540, 1}, {1180, 760, 1}, {1920, 1040, 1}, {1650, 1060, 1.5}, {2560, 1400, 2}, {640, 420, 1}}
	for _, sz := range sizes {
		for _, dark := range []int{1, 2} {
			for bgs := 0; bgs < 3; bgs++ {
				a, fh, now := newTestApp(t, sz.w, sz.h)
				a.S.Theme, a.S.Background = dark, bgs
				a.S.Records = nil
				fakeRecords(a.S, *now)
				adv(a, now, 50*time.Millisecond)
				a.Render()
				for _, m := range []int{modalSettings, modalStats, modalHelp} {
					a.openModal(m)
					for i := 0; i < 30; i++ {
						adv(a, now, 16*time.Millisecond)
					}
					a.Render()
					// tüm tıklanabilir öğeleri dene
					for _, h := range append([]Hit(nil), a.hits...) {
						x, y := int((h.X0+h.X1)/2), int((h.Y0+h.Y1)/2)
						a.MouseMove(x, y)
						a.Render()
						if h.ID != idScrim && h.ID != idModalClose && h.ID != idHelpLink && h.ID != idSettings && h.ID != idStats {
							a.MouseDown(x, y)
							a.MouseUp(x, y)
						}
						adv(a, now, 16*time.Millisecond)
					}
					a.closeModal()
					for i := 0; i < 30; i++ {
						adv(a, now, 16*time.Millisecond)
					}
					a.Render()
				}
				for _, k := range []int{'1', '2', '3', 0x26, 0x28, 'D', 'M', 'P', 0x70, 0x1B, 'C'} {
					a.KeyDown(k, false, false)
					adv(a, now, 16*time.Millisecond)
					a.Render()
				}
				if !a.compact || !fh.compact {
					t.Fatal("C kompakt moda geçmeli")
				}
				a.Resize(300, 100, make([]uint32, 300*100), sz.u)
				a.Render()
				a.KeyDown(0x1B, false, false)
				a.Resize(sz.w, sz.h, make([]uint32, sz.w*sz.h), sz.u)
				a.Render()
				if fh.drawn == 0 {
					t.Fatal("yazı çizilmedi")
				}
			}
		}
	}
}

func TestIdleNoFrames(t *testing.T) {
	a, _, now := newTestApp(t, 1000, 700)
	for i := 0; i < 120; i++ {
		adv(a, now, 16*time.Millisecond)
		if a.WantsFrame() {
			a.Render()
		}
	}
	frames := 0
	for i := 0; i < 300; i++ {
		adv(a, now, 16*time.Millisecond)
		if a.WantsFrame() {
			frames++
			a.Render()
		}
	}
	if frames > 0 {
		t.Fatalf("boşta %d kare çizildi; boşta çizim olmamalı", frames)
	}
	a.toggle()
	frames = 0
	for i := 0; i < 125; i++ { // ~2 sn
		adv(a, now, 16*time.Millisecond)
		if a.WantsFrame() {
			frames++
			a.Render()
		}
	}
	if frames > 40 {
		t.Fatalf("çalışırken 2 sn'de %d kare; 4 fps civarı beklenir", frames)
	}
}
