//go:build darwin && cgo

package main

/*
#cgo LDFLAGS: -framework CoreServices -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <CoreServices/CoreServices.h>

// DCSCopyTextDefinition is the one call: the definition of the word in
// the user's default dictionary, as plain text, or NULL when there is
// none. It is declared in DictionaryServices, part of CoreServices.
static char *define(const char *word) {
	CFStringRef s = CFStringCreateWithCString(NULL, word, kCFStringEncodingUTF8);
	if (s == NULL) return NULL;
	CFRange range = CFRangeMake(0, CFStringGetLength(s));
	CFStringRef def = DCSCopyTextDefinition(NULL, s, range);
	CFRelease(s);
	if (def == NULL) return NULL;
	CFIndex max = CFStringGetMaximumSizeForEncoding(CFStringGetLength(def), kCFStringEncodingUTF8) + 1;
	char *buf = malloc(max);
	if (buf != NULL && !CFStringGetCString(def, buf, max, kCFStringEncodingUTF8)) {
		free(buf);
		buf = NULL;
	}
	CFRelease(def);
	return buf;
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

func define(word string) (string, error) {
	cw := C.CString(word)
	defer C.free(unsafe.Pointer(cw))
	out := C.define(cw)
	if out == nil {
		return "", errors.New("no definition for " + word)
	}
	defer C.free(unsafe.Pointer(out))
	return C.GoString(out), nil
}
