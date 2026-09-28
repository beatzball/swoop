package main

import (
	"fmt"
	"math"
)

// Rect is a frame in global screen points, origin at the top left of the
// main display, y growing down. The Accessibility API speaks this way, so
// the displays are converted to it once and every sum here stays in it.
type Rect struct {
	X, Y, W, H float64
}

func (r Rect) String() string {
	return fmt.Sprintf("%g,%g %g×%g", r.X, r.Y, r.W, r.H)
}

func (r Rect) center() (float64, float64) { return r.X + r.W/2, r.Y + r.H/2 }

// Display is one screen: its whole frame, and the part a window may use,
// which leaves out the menu bar and the Dock.
type Display struct {
	Frame, Visible Rect
}

// Command is one row: what it is called and how it places a window.
type Command struct {
	ID, Title, Subtitle string
}

// commands are the rows, in the order the launcher lists them. The ids are
// what `run` and `preview` take.
var commands = []Command{
	{"left-half", "Left Half", "Move the window to the left half of its display"},
	{"right-half", "Right Half", "Move the window to the right half of its display"},
	{"top-half", "Top Half", "Move the window to the top half of its display"},
	{"bottom-half", "Bottom Half", "Move the window to the bottom half of its display"},
	{"first-third", "First Third", "The first third of the display"},
	{"center-third", "Center Third", "The middle third of the display"},
	{"last-third", "Last Third", "The last third of the display"},
	{"first-two-thirds", "First Two Thirds", "The first two thirds of the display"},
	{"last-two-thirds", "Last Two Thirds", "The last two thirds of the display"},
	{"top-left-quarter", "Top Left Quarter", "The top left quarter of the display"},
	{"top-right-quarter", "Top Right Quarter", "The top right quarter of the display"},
	{"bottom-left-quarter", "Bottom Left Quarter", "The bottom left quarter of the display"},
	{"bottom-right-quarter", "Bottom Right Quarter", "The bottom right quarter of the display"},
	{"maximize", "Maximize", "Fill the display, not full screen"},
	{"almost-maximize", "Almost Maximize", "Nine tenths of the display, centered"},
	{"reasonable-size", "Reasonable Size", "Six tenths of the display, centered"},
	{"center", "Center", "Center the window, same size"},
	{"next-display", "Next Display", "Move the window to the next display"},
	{"previous-display", "Previous Display", "Move the window to the previous display"},
}

func findCommand(id string) (Command, bool) {
	for _, c := range commands {
		if c.ID == id {
			return c, true
		}
	}
	return Command{}, false
}

// displayOf is the display a window is on: the one it covers most, or,
// when it covers none (moved off every screen), the one nearest its
// center. -1 only when there are no displays.
func displayOf(win Rect, ds []Display) int {
	best, bestArea := -1, 0.0
	for i, d := range ds {
		if a := overlap(win, d.Frame); a > bestArea {
			best, bestArea = i, a
		}
	}
	if best >= 0 {
		return best
	}
	cx, cy := win.center()
	bestDist := math.Inf(1)
	for i, d := range ds {
		dx, dy := d.Frame.center()
		if dist := math.Hypot(cx-dx, cy-dy); dist < bestDist {
			best, bestDist = i, dist
		}
	}
	return best
}

func overlap(a, b Rect) float64 {
	w := math.Min(a.X+a.W, b.X+b.W) - math.Max(a.X, b.X)
	h := math.Min(a.Y+a.H, b.Y+b.H) - math.Max(a.Y, b.Y)
	if w <= 0 || h <= 0 {
		return 0
	}
	return w * h
}

// Place is the frame a command gives a window, and the display it lands
// on. It is the whole of the arithmetic, kept apart from the calls that
// read and move windows so it is tested on every OS.
func Place(id string, win Rect, ds []Display) (Rect, int, error) {
	if len(ds) == 0 {
		return Rect{}, -1, fmt.Errorf("no displays")
	}
	i := displayOf(win, ds)
	v := ds[i].Visible
	// Thirds run along the long side: across a landscape display, down a
	// portrait one, where three columns would be too narrow to use.
	portrait := v.H > v.W
	third := func(from, span int) Rect {
		if portrait {
			return rows(v, 3, from, span)
		}
		return cols(v, 3, from, span)
	}
	var r Rect
	switch id {
	case "left-half":
		r = cols(v, 2, 0, 1)
	case "right-half":
		r = cols(v, 2, 1, 1)
	case "top-half":
		r = rows(v, 2, 0, 1)
	case "bottom-half":
		r = rows(v, 2, 1, 1)
	case "first-third":
		r = third(0, 1)
	case "center-third":
		r = third(1, 1)
	case "last-third":
		r = third(2, 1)
	case "first-two-thirds":
		r = third(0, 2)
	case "last-two-thirds":
		r = third(1, 2)
	case "top-left-quarter":
		r = quarter(v, 0, 0)
	case "top-right-quarter":
		r = quarter(v, 1, 0)
	case "bottom-left-quarter":
		r = quarter(v, 0, 1)
	case "bottom-right-quarter":
		r = quarter(v, 1, 1)
	case "maximize":
		r = v
	case "almost-maximize":
		r = centered(v, math.Round(v.W*0.9), math.Round(v.H*0.9))
	case "reasonable-size":
		r = centered(v, math.Round(v.W*0.6), math.Round(v.H*0.6))
	case "center":
		r = centered(v, math.Min(win.W, v.W), math.Min(win.H, v.H))
	case "next-display", "previous-display":
		step := 1
		if id == "previous-display" {
			step = len(ds) - 1
		}
		j := (i + step) % len(ds)
		return carry(win, ds[i].Visible, ds[j].Visible), j, nil
	default:
		return Rect{}, -1, fmt.Errorf("no window command %q", id)
	}
	return r, i, nil
}

// edge is the k-th of n cuts across a length, rounded to a whole point,
// so the pieces of one display meet with no gap and no overlap.
func edge(start, length float64, n, k int) float64 {
	return start + math.Round(length*float64(k)/float64(n))
}

// cols is span columns of n, starting at column from.
func cols(v Rect, n, from, span int) Rect {
	x0, x1 := edge(v.X, v.W, n, from), edge(v.X, v.W, n, from+span)
	return Rect{x0, v.Y, x1 - x0, v.H}
}

// rows is span rows of n, starting at row from.
func rows(v Rect, n, from, span int) Rect {
	y0, y1 := edge(v.Y, v.H, n, from), edge(v.Y, v.H, n, from+span)
	return Rect{v.X, y0, v.W, y1 - y0}
}

func quarter(v Rect, col, row int) Rect {
	c, r := cols(v, 2, col, 1), rows(v, 2, row, 1)
	return Rect{c.X, r.Y, c.W, r.H}
}

func centered(v Rect, w, h float64) Rect {
	return Rect{v.X + math.Round((v.W-w)/2), v.Y + math.Round((v.H-h)/2), w, h}
}

// carry moves a window from one display to another, keeping where it sat
// and how much of the display it took, in proportion: a left half stays a
// left half on a display of another size. A window larger than the
// display it came from is clamped first, so it still fits.
func carry(win, from, to Rect) Rect {
	w, h := math.Min(win.W, from.W), math.Min(win.H, from.H)
	x := math.Max(from.X, math.Min(win.X, from.X+from.W-w))
	y := math.Max(from.Y, math.Min(win.Y, from.Y+from.H-h))
	sx, sy := to.W/from.W, to.H/from.H
	return Rect{
		X: to.X + math.Round((x-from.X)*sx),
		Y: to.Y + math.Round((y-from.Y)*sy),
		W: math.Round(w * sx),
		H: math.Round(h * sy),
	}
}
