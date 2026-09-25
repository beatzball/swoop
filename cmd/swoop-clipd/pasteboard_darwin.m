// The three pasteboard calls the watcher needs, in Objective-C because
// NSPasteboard is an AppKit class. Each runs inside its own autorelease
// pool: the watcher is a long-running loop with no run loop of its own,
// so nothing else would ever drain one.
#import <AppKit/AppKit.h>
#import <CoreGraphics/CoreGraphics.h>
#include <string.h>
#include "pasteboard_darwin.h"

long swoop_pb_change_count(void) {
	@autoreleasepool {
		return (long)[[NSPasteboard generalPasteboard] changeCount];
	}
}

int swoop_pb_concealed(void) {
	@autoreleasepool {
		NSArray<NSPasteboardType> *types = [[NSPasteboard generalPasteboard] types];
		if (types == nil) return 0;
		// The two marks from nspasteboard.org. Concealed is what password
		// managers set on a copied password; Transient is for data that a
		// history should not keep, such as a one-off internal drag.
		return [types containsObject:@"org.nspasteboard.ConcealedType"] ||
		       [types containsObject:@"org.nspasteboard.TransientType"];
	}
}

char *swoop_pb_text(void) {
	@autoreleasepool {
		NSString *s = [[NSPasteboard generalPasteboard] stringForType:NSPasteboardTypeString];
		if (s == nil) return NULL;
		const char *utf8 = [s UTF8String];
		return utf8 ? strdup(utf8) : NULL;
	}
}

// All the types on the pasteboard, one per line, for `swoop-clipd types`:
// the way to learn what a given app marks its copies with.
char *swoop_pb_types(void) {
	@autoreleasepool {
		NSArray<NSPasteboardType> *types = [[NSPasteboard generalPasteboard] types];
		if (types == nil) return strdup("");
		NSString *joined = [types componentsJoinedByString:@"\n"];
		const char *utf8 = [joined UTF8String];
		return utf8 ? strdup(utf8) : strdup("");
	}
}

// The bundle id of the app in front, or "" if none, asked of the window
// server: the owner of the first ordinary window on screen. Not AppKit's
// frontmostApplication, which is what the first version used: in a process
// with no run loop of its own, such as this watcher, that answer is never
// refreshed and kept naming the app that was in front when the watcher
// started. Seen on screen: a password copied in Dashlane's main window was
// attributed to the terminal. The window list is read fresh on every call.
char *swoop_frontmost_app(void) {
	@autoreleasepool {
		CFArrayRef list = CGWindowListCopyWindowInfo(kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements, kCGNullWindowID);
		if (list == NULL) return strdup("");
		char *out = strdup("");
		for (NSDictionary *w in (__bridge NSArray *)list) {
			// Layer 0 is an ordinary window; menu bar items, the dock and
			// overlays sit on other layers and are skipped.
			if ([w[(id)kCGWindowLayer] intValue] != 0) continue;
			pid_t pid = [w[(id)kCGWindowOwnerPID] intValue];
			NSRunningApplication *app = [NSRunningApplication runningApplicationWithProcessIdentifier:pid];
			NSString *bid = app ? [app bundleIdentifier] : nil;
			if (bid != nil) {
				free(out);
				out = strdup([bid UTF8String]);
			}
			break;
		}
		CFRelease(list);
		return out;
	}
}
