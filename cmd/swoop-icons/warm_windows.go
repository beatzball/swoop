package main

// startWarmer does nothing here: no icon is ever reported as not cached
// on this OS, so there is nothing to warm.
func startWarmer([]string) error { return nil }

// warm does nothing here, for the same reason.
func warm([]string) error { return nil }
