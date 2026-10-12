//go:build linux

package main

// Yalnızca geliştirme önizlemesi (Windows .exe'ye girmez):
//   go run . shots <çıktı klasörü>   → ekran görüntüleri
//   go run . icon  <çıktı klasörü>   → uygulama simgesi PNG'leri

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type previewHost struct {
	app    *App
	widths map[string]map[string]float64
	tools  string
	tmp    string
	dark   bool
}

func fontKey(f Font) string {
	w := 400
	switch {
	case f.Weight <= 350:
		w = 300
	case f.Weight >= 650:
		w = 700
	case f.Weight >= 550:
		w = 600
	}
	if f.Face == faceDisp {
		if w == 700 {
			w = 600
		}
		return fmt.Sprintf("d%d", w)
	}
	if w == 300 {
		w = 400
	}
	return fmt.Sprintf("t%d", w)
}

func (h *previewHost) Measure(f Font, s string) int {
	tbl := h.widths[fontKey(f)]
	sum := 0.0
	for _, r := range s {
		w, ok := tbl[string(r)]
		if !ok {
			w = tbl["n"]
		}
		sum += w
	}
	return int(math.Round(sum * float64(f.Size) / 100))
}

func (h *previewHost) DrawText(cmds []TextCmd) {
	c := h.app.c
	raw := make([]byte, c.W*c.H*3)
	for i, p := range c.Pix {
		raw[i*3], raw[i*3+1], raw[i*3+2] = byte(p>>16), byte(p>>8), byte(p)
	}
	type jc struct {
		S    string `json:"s"`
		X    int    `json:"x"`
		Y    int    `json:"y"`
		A    int8   `json:"a"`
		K    string `json:"k"`
		Size int    `json:"size"`
		C    [3]int `json:"c"`
		Clip bool   `json:"clip"`
		CX0  int    `json:"cx0"`
		CX1  int    `json:"cx1"`
	}
	var list []jc
	for _, t := range cmds {
		list = append(list, jc{t.S, t.X, t.Y, t.Align, fontKey(t.F), t.F.Size, t.C.q(), t.Clip, t.CX0, t.CX1})
	}
	rp := filepath.Join(h.tmp, "frame.raw")
	jp := filepath.Join(h.tmp, "frame.json")
	os.WriteFile(rp, raw, 0o644)
	b, _ := json.Marshal(map[string]any{"w": c.W, "h": c.H, "cmds": list})
	os.WriteFile(jp, b, 0o644)
	if out, err := exec.Command("python3", filepath.Join(h.tools, "drawtext.py"), rp, jp).CombinedOutput(); err != nil {
		fmt.Println("drawtext:", err, string(out))
		return
	}
	raw, _ = os.ReadFile(rp)
	for i := range c.Pix {
		c.Pix[i] = pack(int(raw[i*3]), int(raw[i*3+1]), int(raw[i*3+2]))
	}
}

func (h *previewHost) PlaySound([]byte)         {}
func (h *previewHost) Notify(t, b string)       { fmt.Println("bildirim:", t, "|", b) }
func (h *previewHost) Attention()               {}
func (h *previewHost) SetTopmost(bool)          {}
func (h *previewHost) SetCompact(bool)          {}
func (h *previewHost) SystemDark() bool         { return h.dark }
func (h *previewHost) ReduceMotion() bool       { return false }
func (h *previewHost) SetTitle(string)          {}
func (h *previewHost) SetProgress(int, float64) {}
func (h *previewHost) SetTrayTip(string)        {}
func (h *previewHost) SetChrome(bool, RGB, RGB) {}
func (h *previewHost) Clipboard() string        { return "" }

func savePNG(c *Canvas, name string) {
	img := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	for i, p := range c.Pix {
		img.Pix[i*4+0] = uint8(p >> 16)
		img.Pix[i*4+1] = uint8(p >> 8)
		img.Pix[i*4+2] = uint8(p)
		img.Pix[i*4+3] = 255
	}
	f, _ := os.Create(name)
	png.Encode(f, img)
	f.Close()
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("kullanım: go run . shots|icon <klasör>")
		return
	}
	out := os.Args[2]
	os.MkdirAll(out, 0o755)
	if os.Args[1] == "icon" {
		for _, s := range []int{16, 20, 24, 32, 40, 48, 64, 128, 256} {
			c := NewCanvas(s, s)
			DrawAppIcon(c)
			img := image.NewNRGBA(image.Rect(0, 0, s, s))
			for i, p := range c.Pix {
				img.Pix[i*4+0] = uint8(p >> 16)
				img.Pix[i*4+1] = uint8(p >> 8)
				img.Pix[i*4+2] = uint8(p)
				img.Pix[i*4+3] = uint8(p >> 24)
			}
			f, _ := os.Create(filepath.Join(out, fmt.Sprintf("icon_%d.png", s)))
			png.Encode(f, img)
			f.Close()
		}
		return
	}

	wd, _ := os.Getwd()
	tools := filepath.Join(wd, "tools")
	tmp := os.Getenv("SNAP_TMP")
	if tmp == "" {
		tmp = os.TempDir()
	}
	widths := map[string]map[string]float64{}
	b, err := os.ReadFile(filepath.Join(tmp, "widths.json"))
	if err != nil {
		fmt.Println("widths.json yok:", err)
		return
	}
	json.Unmarshal(b, &widths)

	type shot struct {
		name  string
		w, h  int
		u     float64
		dark  bool
		setup func(a *App, now *time.Time)
	}
	base := time.Date(2026, 10, 12, 14, 30, 0, 0, time.Local)
	step := func(a *App, now *time.Time, sec float64) {
		for i := 0; i < int(sec/0.016); i++ {
			*now = now.Add(16 * time.Millisecond)
			a.Update(*now)
		}
	}
	shots := []shot{
		{"01_koyu_bosta", 1180, 760, 1, true, func(a *App, now *time.Time) {}},
		{"02_acik_calisiyor", 1180, 760, 1, false, func(a *App, now *time.Time) {
			a.S.Task = "Devre analizi ödevi"
			a.M.Cycle = 2
			a.toggle()
			step(a, now, 7*60+12)
			a.MouseMove(590, 700)
			step(a, now, 1.0)
		}},
		{"03_koyu_ayarlar", 1180, 760, 1, true, func(a *App, now *time.Time) {
			a.openModal(modalSettings)
			step(a, now, 0.6)
		}},
		{"04_acik_istatistik", 1180, 760, 1, false, func(a *App, now *time.Time) {
			a.openModal(modalStats)
			step(a, now, 0.6)
			a.Render()
			// 5. çubuğun üzerine gel
			for _, h := range a.hits {
				if h.ID == idChartBar+4 {
					a.MouseMove(int((h.X0+h.X1)/2), int(h.Y1-60))
				}
			}
			step(a, now, 0.3)
		}},
		{"05_koyu_kisayollar", 1180, 760, 1, true, func(a *App, now *time.Time) {
			a.openModal(modalHelp)
			step(a, now, 0.6)
		}},
		{"06_koyu_sehir_mola", 1180, 760, 1, true, func(a *App, now *time.Time) {
			a.S.Background = 2
			a.M.Cycle = 1
			a.setMode(Short)
			a.toggle()
			step(a, now, 62)
		}},
		{"07_acik_sehir_uzun", 1180, 760, 1, false, func(a *App, now *time.Time) {
			a.S.Background = 2
			a.M.Cycle = 4
			a.setMode(Long)
			step(a, now, 0.5)
			a.Render()
			for _, h := range a.hits {
				if h.ID == idSettings {
					a.MouseMove(int((h.X0+h.X1)/2), int((h.Y0+h.Y1)/2))
				}
			}
			step(a, now, 1.0)
		}},
		{"08_kucuk_pencere", 760, 540, 1, true, func(a *App, now *time.Time) {
			a.beginEdit()
			a.insertText("Sinyaller ve sistemler, 3. bölüm")
			step(a, now, 0.2)
		}},
		{"09_yuksek_dpi", 1650, 1060, 1.5, false, func(a *App, now *time.Time) {
			a.S.Task = "TEG proje raporu"
			a.toggle()
			step(a, now, 3)
			a.toggle()
			a.MouseMove(825, 300)
			step(a, now, 0.5)
		}},
		{"10_kompakt_koyu", 300, 100, 1, true, func(a *App, now *time.Time) {
			a.compact = true
			a.toggle()
			step(a, now, 400)
		}},
		{"11_kompakt_acik", 300, 100, 1, false, func(a *App, now *time.Time) {
			a.compact = true
			a.setMode(Short)
			step(a, now, 0.6)
		}},
		{"12_koyu_ayarlar_kucuk", 760, 540, 1, true, func(a *App, now *time.Time) {
			a.openModal(modalSettings)
			step(a, now, 0.6)
		}},
		{"13_koyu_bitis_toast", 1180, 760, 1, true, func(a *App, now *time.Time) {
			a.S.FocusMin = 1
			a.applyDurations()
			a.toggle()
			step(a, now, 60.6)
		}},
	}
	only := ""
	if len(os.Args) > 3 {
		only = os.Args[3]
	}
	for _, sh := range shots {
		if only != "" && !contains(sh.name, only) {
			continue
		}
		h := &previewHost{widths: widths, tools: tools, tmp: tmp, dark: sh.dark}
		cfg := filepath.Join(tmp, "cfg_"+sh.name+".json")
		os.Remove(cfg)
		a := NewApp(h, cfg)
		h.app = a
		fakeRecords(a.S, base)
		a.lastT = base
		now := base
		pix := make([]uint32, sh.w*sh.h)
		a.Resize(sh.w, sh.h, pix, sh.u)
		step(a, &now, 1.2)
		sh.setup(a, &now)
		t0 := time.Now()
		a.Render()
		fmt.Printf("%-24s %4dx%-4d çizim %v\n", sh.name, sh.w, sh.h, time.Since(t0).Round(time.Millisecond))
		savePNG(a.c, filepath.Join(out, sh.name+".png"))
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
