//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework CoreGraphics
#include <stdlib.h>
#include "pasteboard_darwin.h"
*/
import "C"

import "unsafe"

// pasteboard is the macOS general pasteboard, through AppKit.
type pasteboard struct{}

func openPasteboard() (pasteboard, error) { return pasteboard{}, nil }

// ChangeCount goes up by one every time anything is copied. Comparing it
// is how the watcher notices a change without reading the content.
func (pasteboard) ChangeCount() int64 { return int64(C.swoop_pb_change_count()) }

// Concealed reports whether the current content carries the marks that
// password managers and other careful apps put on secrets or one-shot
// data: org.nspasteboard.ConcealedType and org.nspasteboard.TransientType.
// Such content is never stored.
func (pasteboard) Concealed() bool { return C.swoop_pb_concealed() != 0 }

// Text returns the current content as a string, if it has one.
func (pasteboard) Text() (string, bool) {
	p := C.swoop_pb_text()
	if p == nil {
		return "", false
	}
	defer C.free(unsafe.Pointer(p))
	return C.GoString(p), true
}

// Types lists the pasteboard's current types, one per line.
func (pasteboard) Types() string {
	p := C.swoop_pb_types()
	defer C.free(unsafe.Pointer(p))
	return C.GoString(p)
}

// Frontmost is the bundle id of the app in front right now.
func (pasteboard) Frontmost() string {
	p := C.swoop_frontmost_app()
	defer C.free(unsafe.Pointer(p))
	return C.GoString(p)
}
