package main

import "testing"

// A 1440×900 laptop with a 25-point menu bar, and a 1920×1080 display to
// its right, a little higher, with no menu bar of its own.
var (
	laptop = Display{Frame: Rect{0, 0, 1440, 900}, Visible: Rect{0, 25, 1440, 875}}
	wide   = Display{Frame: Rect{1440, -200, 1920, 1080}, Visible: Rect{1440, -200, 1920, 1080}}
	// Rotated: taller than wide, so thirds are rows.
	tall = Display{Frame: Rect{-1080, 0, 1080, 1920}, Visible: Rect{-1080, 0, 1080, 1920}}
)

func TestPlaceOnOneDisplay(t *testing.T) {
	win := Rect{100, 100, 800, 600}
	cases := map[string]Rect{
		"left-half":            {0, 25, 720, 875},
		"right-half":           {720, 25, 720, 875},
		"top-half":             {0, 25, 1440, 438},
		"bottom-half":          {0, 463, 1440, 437},
		"first-third":          {0, 25, 480, 875},
		"center-third":         {480, 25, 480, 875},
		"last-third":           {960, 25, 480, 875},
		"first-two-thirds":     {0, 25, 960, 875},
		"last-two-thirds":      {480, 25, 960, 875},
		"top-left-quarter":     {0, 25, 720, 438},
		"top-right-quarter":    {720, 25, 720, 438},
		"bottom-left-quarter":  {0, 463, 720, 437},
		"bottom-right-quarter": {720, 463, 720, 437},
		"maximize":             {0, 25, 1440, 875},
		"almost-maximize":      {72, 69, 1296, 788},
		"reasonable-size":      {288, 200, 864, 525},
		"center":               {320, 163, 800, 600},
	}
	for id, want := range cases {
		got, d, err := Place(id, win, []Display{laptop})
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if got != want || d != 0 {
			t.Errorf("%s: got %v on %d, want %v on 0", id, got, d, want)
		}
	}
}

// Every command has a place, and every place is inside the visible area.
func TestEveryCommandFits(t *testing.T) {
	ds := []Display{laptop, wide, tall}
	for _, win := range []Rect{{100, 100, 800, 600}, {1500, 0, 3000, 3000}, {-900, 300, 200, 200}} {
		for _, c := range commands {
			got, d, err := Place(c.ID, win, ds)
			if err != nil {
				t.Fatalf("%s: %v", c.ID, err)
			}
			v := ds[d].Visible
			if got.W <= 0 || got.H <= 0 || got.X < v.X || got.Y < v.Y || got.X+got.W > v.X+v.W || got.Y+got.H > v.Y+v.H {
				t.Errorf("%s from %v: %v is not inside display %d's %v", c.ID, win, got, d, v)
			}
		}
	}
}

// Halves and thirds of an odd width meet exactly: no gap, no overlap.
func TestPiecesTile(t *testing.T) {
	odd := Display{Frame: Rect{0, 0, 1001, 701}, Visible: Rect{0, 0, 1001, 701}}
	ds := []Display{odd}
	win := Rect{10, 10, 100, 100}
	left, _, _ := Place("left-half", win, ds)
	right, _, _ := Place("right-half", win, ds)
	if left.X+left.W != right.X || left.W+right.W != 1001 {
		t.Errorf("halves do not tile: %v %v", left, right)
	}
	a, _, _ := Place("first-third", win, ds)
	b, _, _ := Place("center-third", win, ds)
	c, _, _ := Place("last-third", win, ds)
	if a.X+a.W != b.X || b.X+b.W != c.X || c.X+c.W != 1001 {
		t.Errorf("thirds do not tile: %v %v %v", a, b, c)
	}
	two, _, _ := Place("first-two-thirds", win, ds)
	if two.X+two.W != c.X {
		t.Errorf("first two thirds do not meet the last third: %v %v", two, c)
	}
}

func TestThirdsOnAPortraitDisplay(t *testing.T) {
	got, _, _ := Place("first-third", Rect{-1000, 100, 500, 500}, []Display{tall})
	if want := (Rect{-1080, 0, 1080, 640}); got != want {
		t.Errorf("first third of a portrait display: got %v, want %v", got, want)
	}
}

// The window's display is the one it covers most, even when its top left
// corner is on another.
func TestDisplayOf(t *testing.T) {
	ds := []Display{laptop, wide}
	if d := displayOf(Rect{1300, 100, 800, 600}, ds); d != 1 {
		t.Errorf("mostly on the wide display: got %d", d)
	}
	if d := displayOf(Rect{5000, 5000, 100, 100}, ds); d != 1 {
		t.Errorf("off every screen, nearest the wide one: got %d", d)
	}
	if d := displayOf(Rect{}, nil); d != -1 {
		t.Errorf("no displays: got %d", d)
	}
}

func TestNextAndPreviousDisplay(t *testing.T) {
	ds := []Display{laptop, wide, tall}
	// The laptop's left half becomes the wide display's left half.
	half := Rect{0, 25, 720, 875}
	got, d, _ := Place("next-display", half, ds)
	if want := (Rect{1440, -200, 960, 1080}); got != want || d != 1 {
		t.Errorf("next: got %v on %d, want %v on 1", got, d, want)
	}
	// Previous from the first wraps to the last.
	_, d, _ = Place("previous-display", half, ds)
	if d != 2 {
		t.Errorf("previous from the first display: got %d, want 2", d)
	}
	// One display: next is the same display, and the window stays put.
	got, d, _ = Place("next-display", half, []Display{laptop})
	if got != half || d != 0 {
		t.Errorf("next with one display moved the window: %v on %d", got, d)
	}
}

func TestPlaceRefuses(t *testing.T) {
	if _, _, err := Place("left-half", Rect{}, nil); err == nil {
		t.Error("no displays should be an error")
	}
	if _, _, err := Place("sideways", Rect{}, []Display{laptop}); err == nil {
		t.Error("an unknown command should be an error")
	}
}
