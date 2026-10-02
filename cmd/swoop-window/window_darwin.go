//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework ApplicationServices
#import <AppKit/AppKit.h>
#import <ApplicationServices/ApplicationServices.h>
#include <stdlib.h>
#include <string.h>

static int sw_trusted(void) { return AXIsProcessTrusted() ? 1 : 0; }

// sw_ask asks with the prompt option. That shows the system's own dialog
// and, more usefully, puts the app that is responsible for this process
// into the Accessibility list, so granting it is one switch rather than
// finding a binary with the + button.
static void sw_ask(void) {
	@autoreleasepool {
		NSDictionary *opts = @{(__bridge id)kAXTrustedCheckOptionPrompt: @YES};
		AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)opts);
	}
}

static char *sw_strdup(NSString *s) {
	if (s == nil) return NULL;
	const char *c = [s UTF8String];
	return c ? strdup(c) : NULL;
}

typedef struct {
	AXUIElementRef win;
	int pid;
	double x, y, w, h;
	char *app;
	char *title;
} sw_window;

// sw_focused finds the focused window of the app in front. The frame's
// panel is non-activating, so that app is still the one the user was in.
// NSWorkspace answers correctly here because this is a fresh process: it
// only goes stale in a long-running one with no run loop.
//
// Returns 0, or: 1 no app in front, 2 no window, 3 the frame unreadable,
// 4 Accessibility refused, 5 the app did not answer in time.
static int sw_focused(sw_window *out) {
	@autoreleasepool {
		memset(out, 0, sizeof *out);
		NSRunningApplication *front = [[NSWorkspace sharedWorkspace] frontmostApplication];
		if (front == nil) return 1;
		out->pid = front.processIdentifier;
		out->app = sw_strdup(front.localizedName);
		AXUIElementRef app = AXUIElementCreateApplication(out->pid);
		// An app that is busy or hung answers nothing; the default wait
		// is six seconds a call, and the preview runs on every cursor
		// move. One second, and the row says the window would not answer.
		AXUIElementSetMessagingTimeout(app, 1.0);
		CFTypeRef win = NULL;
		AXError err = AXUIElementCopyAttributeValue(app, kAXFocusedWindowAttribute, &win);
		if (err != kAXErrorSuccess || win == NULL) {
			err = AXUIElementCopyAttributeValue(app, kAXMainWindowAttribute, &win);
		}
		CFRelease(app);
		if (err == kAXErrorAPIDisabled) return 4;
		if (err == kAXErrorCannotComplete) return 5;
		if (err != kAXErrorSuccess || win == NULL) return 2;
		out->win = (AXUIElementRef)win;
		AXUIElementSetMessagingTimeout(out->win, 1.0);

		CFTypeRef pos = NULL, size = NULL, title = NULL;
		CGPoint p;
		CGSize s;
		int ok = AXUIElementCopyAttributeValue(out->win, kAXPositionAttribute, &pos) == kAXErrorSuccess &&
			AXUIElementCopyAttributeValue(out->win, kAXSizeAttribute, &size) == kAXErrorSuccess &&
			AXValueGetValue(pos, kAXValueTypeCGPoint, &p) && AXValueGetValue(size, kAXValueTypeCGSize, &s);
		if (pos) CFRelease(pos);
		if (size) CFRelease(size);
		if (!ok) return 3;
		out->x = p.x; out->y = p.y; out->w = s.width; out->h = s.height;
		if (AXUIElementCopyAttributeValue(out->win, kAXTitleAttribute, &title) == kAXErrorSuccess && title) {
			if (CFGetTypeID(title) == CFStringGetTypeID()) out->title = sw_strdup((__bridge NSString *)title);
			CFRelease(title);
		}
		return 0;
	}
}

// sw_screens writes up to max displays, eight numbers each: the frame and
// the visible frame, in the Accessibility API's coordinates. AppKit puts
// the origin at the bottom left of the main display with y up; the
// Accessibility API puts it at the top left with y down. The main display
// is screens[0], and its height is the flip.
static int sw_screens(double *out, int max) {
	@autoreleasepool {
		NSArray<NSScreen *> *screens = [NSScreen screens];
		if (screens.count == 0) return 0;
		CGFloat top = screens[0].frame.size.height;
		int n = 0;
		for (NSScreen *s in screens) {
			if (n == max) break;
			NSRect f = s.frame, v = s.visibleFrame;
			double *o = out + n * 8;
			o[0] = f.origin.x; o[1] = top - f.origin.y - f.size.height; o[2] = f.size.width; o[3] = f.size.height;
			o[4] = v.origin.x; o[5] = top - v.origin.y - v.size.height; o[6] = v.size.width; o[7] = v.size.height;
			n++;
		}
		return n;
	}
}

// sw_set moves and sizes a window. Size, position, size: a window moving
// to a smaller display is first held back by the edge of the one it is
// on, so the second size is the one that sticks.
//
// Apps that turn on AXEnhancedUserInterface for assistive tools animate
// every change, slowly and sometimes to the wrong place; it is turned off
// for the move and put back after, as window managers do.
static int sw_set(AXUIElementRef win, int pid, double x, double y, double w, double h) {
	AXUIElementRef app = AXUIElementCreateApplication(pid);
	CFStringRef enhanced = CFSTR("AXEnhancedUserInterface");
	CFTypeRef was = NULL;
	int restore = AXUIElementCopyAttributeValue(app, enhanced, &was) == kAXErrorSuccess && was == kCFBooleanTrue;
	if (was) CFRelease(was);
	if (restore) AXUIElementSetAttributeValue(app, enhanced, kCFBooleanFalse);

	CGPoint p = CGPointMake(x, y);
	CGSize s = CGSizeMake(w, h);
	AXValueRef pv = AXValueCreate(kAXValueTypeCGPoint, &p);
	AXValueRef sv = AXValueCreate(kAXValueTypeCGSize, &s);
	AXUIElementSetAttributeValue(win, kAXSizeAttribute, sv);
	AXError err = AXUIElementSetAttributeValue(win, kAXPositionAttribute, pv);
	AXError err2 = AXUIElementSetAttributeValue(win, kAXSizeAttribute, sv);
	CFRelease(pv);
	CFRelease(sv);

	if (restore) AXUIElementSetAttributeValue(app, enhanced, kCFBooleanTrue);
	CFRelease(app);
	if (err == kAXErrorSuccess) err = err2;
	return (int)err;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"os/exec"
	"unsafe"
)

// settingsPane is System Settings, Privacy & Security, Accessibility.
const settingsPane = "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility"

func trusted() bool { return C.sw_trusted() != 0 }

// askTrust puts the frame in the Accessibility list and opens the pane, so
// the first run without the permission shows where to grant it.
func askTrust() error {
	C.sw_ask()
	return exec.Command("open", settingsPane).Run()
}

// Window is the focused window of the app in front.
type Window struct {
	App, Title string
	Frame      Rect
	pid        C.int
	ref        C.AXUIElementRef
}

func focused() (Window, error) {
	var w C.sw_window
	code := C.sw_focused(&w)
	defer C.free(unsafe.Pointer(w.app))
	defer C.free(unsafe.Pointer(w.title))
	app := C.GoString(w.app)
	switch code {
	case 0:
	case 1:
		return Window{}, errors.New("no app is in front")
	case 2:
		return Window{}, fmt.Errorf("%s has no window to move", app)
	case 3:
		return Window{}, fmt.Errorf("the window of %s does not say where it is", app)
	case 5:
		return Window{}, fmt.Errorf("%s did not answer; it may be busy", app)
	default:
		return Window{}, errNotTrusted
	}
	return Window{
		App:   app,
		Title: C.GoString(w.title),
		Frame: Rect{float64(w.x), float64(w.y), float64(w.w), float64(w.h)},
		pid:   w.pid,
		ref:   w.win,
	}, nil
}

func displays() ([]Display, error) {
	const max = 16
	var buf [max * 8]C.double
	n := int(C.sw_screens(&buf[0], max))
	if n == 0 {
		return nil, errors.New("no displays")
	}
	ds := make([]Display, n)
	for i := range ds {
		o := buf[i*8:]
		ds[i] = Display{
			Frame:   Rect{float64(o[0]), float64(o[1]), float64(o[2]), float64(o[3])},
			Visible: Rect{float64(o[4]), float64(o[5]), float64(o[6]), float64(o[7])},
		}
	}
	return ds, nil
}

func move(w Window, r Rect) error {
	switch code := C.sw_set(w.ref, w.pid, C.double(r.X), C.double(r.Y), C.double(r.W), C.double(r.H)); code {
	case C.kAXErrorSuccess:
		return nil
	case C.kAXErrorAPIDisabled:
		return errNotTrusted
	default:
		return fmt.Errorf("%s would not let its window move (AXError %d)", w.App, int(code))
	}
}
