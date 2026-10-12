package main

import (
	"fmt"
	"math"
	"time"
)

type Mode int

const (
	Focus Mode = iota
	Short
	Long
)

var modeName = [3]string{"Odak", "Kısa mola", "Uzun mola"}

// Model: saf zamanlayıcı durumu. Kayıt, bildirim gibi yan etkiler App'tedir.
type Model struct {
	Dur       [3]time.Duration
	LongEvery int
	Mode      Mode
	Remaining time.Duration
	Running   bool
	EndAt     time.Time
	Cycle     int // bu sette tamamlanan odak sayısı
}

func NewModel(d [3]time.Duration, longEvery int) Model {
	m := Model{Dur: d, LongEvery: longEvery}
	m.Remaining = d[Focus]
	return m
}

// SetDurations: başlamamış oturumun kalan süresini yeni değere çeker.
func (m *Model) SetDurations(d [3]time.Duration, longEvery int) {
	fresh := !m.Running && !m.Started()
	m.Dur = d
	m.LongEvery = longEvery
	if fresh {
		m.Remaining = d[m.Mode]
	}
}

func (m *Model) Start(now time.Time) {
	if m.Running {
		return
	}
	if m.Remaining <= 0 {
		m.Remaining = m.Dur[m.Mode]
	}
	m.EndAt = now.Add(m.Remaining)
	m.Running = true
}

func (m *Model) Pause(now time.Time) {
	if !m.Running {
		return
	}
	m.Remaining = m.EndAt.Sub(now)
	if m.Remaining < 0 {
		m.Remaining = 0
	}
	m.Running = false
}

func (m *Model) Reset() {
	m.Running = false
	m.Remaining = m.Dur[m.Mode]
}

func (m *Model) SetMode(md Mode) {
	m.Mode = md
	m.Reset()
}

// Tick süreyi günceller; süre dolduysa true döner (geçişi App yapar).
func (m *Model) Tick(now time.Time) bool {
	if !m.Running {
		return false
	}
	m.Remaining = m.EndAt.Sub(now)
	if m.Remaining <= 0 {
		m.Remaining = 0
		m.Running = false
		return true
	}
	return false
}

// Advance sıradaki moda geçer (duraklatılmış halde).
func (m *Model) Advance(countFocus bool) {
	switch m.Mode {
	case Focus:
		if countFocus {
			m.Cycle++
		}
		if m.Cycle >= m.LongEvery {
			m.Mode = Long
		} else {
			m.Mode = Short
		}
	case Short:
		m.Mode = Focus
	case Long:
		m.Cycle = 0
		m.Mode = Focus
	}
	m.Running = false
	m.Remaining = m.Dur[m.Mode]
}

func (m *Model) Elapsed() time.Duration {
	e := m.Dur[m.Mode] - m.Remaining
	if e < 0 {
		return 0
	}
	return e
}

func (m *Model) Frac() float64 {
	if m.Dur[m.Mode] <= 0 {
		return 0
	}
	return math.Max(0, math.Min(1, float64(m.Remaining)/float64(m.Dur[m.Mode])))
}

func (m *Model) Secs() int {
	s := int(math.Ceil(m.Remaining.Seconds() - 1e-9))
	if s < 0 {
		s = 0
	}
	return s
}

func (m *Model) Clock() string {
	s := m.Secs()
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}

// Started: süre kısmen harcanmış mı.
func (m *Model) Started() bool { return m.Remaining < m.Dur[m.Mode] }
