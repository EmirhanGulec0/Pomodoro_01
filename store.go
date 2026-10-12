package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Record: biten (veya en az 1 dk sürüp yarıda kalan) bir odak oturumu.
type Record struct {
	End  int64  `json:"end"`
	Sec  int    `json:"sec"`
	Task string `json:"task,omitempty"`
	Full bool   `json:"full"`
}

type Place struct {
	X, Y, W, H int
	Max        bool
	Ok         bool
}

type Settings struct {
	Version    int
	FocusMin   int
	ShortMin   int
	LongMin    int
	LongEvery  int
	AutoBreaks bool
	AutoFocus  bool
	Theme      int // 0 sistem, 1 açık, 2 koyu
	Background int // 0 düz, 1 sade, 2 şehir
	Sound      int // 0 zil, 1 çan, 2 dijital, 3 kapalı
	LastSound  int
	Volume     int // 0..100
	Notify     bool
	Topmost    bool
	Goal       int
	Task       string
	Win        Place
	Mini       Place
	StartMini  bool
	Records    []Record
}

func DefaultSettings() *Settings {
	return &Settings{
		Version: 2, FocusMin: 25, ShortMin: 5, LongMin: 15, LongEvery: 4,
		Theme: 0, Background: 1, Sound: 1, LastSound: 1, Volume: 70,
		Notify: true, Goal: 8,
	}
}

func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (s *Settings) sanitize() {
	s.FocusMin = clampi(s.FocusMin, 1, 180)
	s.ShortMin = clampi(s.ShortMin, 1, 60)
	s.LongMin = clampi(s.LongMin, 1, 90)
	s.LongEvery = clampi(s.LongEvery, 2, 8)
	s.Theme = clampi(s.Theme, 0, 2)
	s.Background = clampi(s.Background, 0, 2)
	s.Sound = clampi(s.Sound, 0, 3)
	s.LastSound = clampi(s.LastSound, 0, 2)
	s.Volume = clampi(s.Volume, 0, 100)
	s.Goal = clampi(s.Goal, 1, 24)
	if r := []rune(s.Task); len(r) > 80 {
		s.Task = string(r[:80])
	}
}

func (s *Settings) Durations() [3]time.Duration {
	return [3]time.Duration{
		time.Duration(s.FocusMin) * time.Minute,
		time.Duration(s.ShortMin) * time.Minute,
		time.Duration(s.LongMin) * time.Minute,
	}
}

func LoadSettings(path string) *Settings {
	s := DefaultSettings()
	if path == "" {
		return s
	}
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, s); err != nil {
			// Bozuk dosyayı yedekle, varsayılanlarla devam et.
			_ = os.Rename(path, path+".bozuk")
			s = DefaultSettings()
		}
	}
	s.sanitize()
	return s
}

// Save: önce geçici dosyaya yazıp yeniden adlandırır (yarım yazılmış dosya olmaz).
func (s *Settings) Save(path string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Settings) AddRecord(r Record) {
	s.Records = append(s.Records, r)
	cut := time.Now().AddDate(-1, -1, 0).Unix()
	i := 0
	for i < len(s.Records) && s.Records[i].End < cut {
		i++
	}
	if len(s.Records)-i > 6000 {
		i = len(s.Records) - 6000
	}
	if i > 0 {
		s.Records = append([]Record(nil), s.Records[i:]...)
	}
}

// ---------------------------------------------------------------------------
// İstatistikler
// ---------------------------------------------------------------------------

type DayStat struct {
	Date  time.Time
	Count int // tamamlanan pomodoro
	Sec   int // toplam odak saniyesi (yarıda kalanlar dahil)
}

func dayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func (s *Settings) dayMap(loc *time.Location) map[time.Time]*DayStat {
	out := map[time.Time]*DayStat{}
	for _, r := range s.Records {
		d := dayStart(time.Unix(r.End, 0).In(loc))
		ds := out[d]
		if ds == nil {
			ds = &DayStat{Date: d}
			out[d] = ds
		}
		ds.Sec += r.Sec
		if r.Full {
			ds.Count++
		}
	}
	return out
}

func (s *Settings) Day(now time.Time) DayStat {
	d := dayStart(now)
	if ds := s.dayMap(now.Location())[d]; ds != nil {
		return *ds
	}
	return DayStat{Date: d}
}

// LastDays: bugün dahil son n gün, eskiden yeniye.
func (s *Settings) LastDays(now time.Time, n int) []DayStat {
	m := s.dayMap(now.Location())
	out := make([]DayStat, n)
	today := dayStart(now)
	for i := 0; i < n; i++ {
		d := today.AddDate(0, 0, -(n - 1 - i))
		if ds := m[d]; ds != nil {
			out[i] = *ds
		} else {
			out[i] = DayStat{Date: d}
		}
	}
	return out
}

// Streak: art arda en az bir pomodoro tamamlanan gün sayısı (bugün henüz yoksa dünden sayar).
func (s *Settings) Streak(now time.Time) int {
	m := s.dayMap(now.Location())
	d := dayStart(now)
	if ds := m[d]; ds == nil || ds.Count == 0 {
		d = d.AddDate(0, 0, -1)
	}
	n := 0
	for {
		ds := m[d]
		if ds == nil || ds.Count == 0 {
			return n
		}
		n++
		d = d.AddDate(0, 0, -1)
	}
}

func (s *Settings) Totals() (count, sec int) {
	for _, r := range s.Records {
		sec += r.Sec
		if r.Full {
			count++
		}
	}
	return
}

// TodayRecords: bugünün oturumları, yeniden eskiye.
func (s *Settings) TodayRecords(now time.Time) []Record {
	d := dayStart(now)
	var out []Record
	for i := len(s.Records) - 1; i >= 0; i-- {
		r := s.Records[i]
		if dayStart(time.Unix(r.End, 0).In(now.Location())).Equal(d) {
			out = append(out, r)
		}
	}
	return out
}

func fmtDur(sec int) string {
	m := sec / 60
	h := m / 60
	switch {
	case h > 0 && m%60 == 0:
		return fmt.Sprintf("%d sa", h)
	case h > 0:
		return fmt.Sprintf("%d sa %d dk", h, m%60)
	}
	return fmt.Sprintf("%d dk", m)
}

var (
	dayShort = [7]string{"Paz", "Pzt", "Sal", "Çar", "Per", "Cum", "Cmt"}
	dayLong  = [7]string{"Pazar", "Pazartesi", "Salı", "Çarşamba", "Perşembe", "Cuma", "Cumartesi"}
	monShort = [12]string{"Oca", "Şub", "Mar", "Nis", "May", "Haz", "Tem", "Ağu", "Eyl", "Eki", "Kas", "Ara"}
)
