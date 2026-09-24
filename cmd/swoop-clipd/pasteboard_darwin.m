// The three pasteboard calls the watcher needs, in Objective-C because
// NSPasteboard is an AppKit class. Each runs inside its own autorelease
// pool: the watcher is a long-running loop with no run loop of its own,
// so nothing else would ever drain one.
#import <AppKit/AppKit.h>
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
