//go:build sharedstore

package checker

import "os"

// concurrentStores reports whether checkers may share a store concurrently,
// in which case its maps take locks. It is set once, before any checker is
// created.
var concurrentStores = os.Getenv("TSGO_SHARED_STORE") == "concurrent"
