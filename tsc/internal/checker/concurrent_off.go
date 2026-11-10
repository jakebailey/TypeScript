//go:build !sharedstore

package checker

// concurrentStores reports whether checkers may share a store concurrently,
// in which case its maps take locks. Only builds with the sharedstore tag
// support it, so that checkers that cannot share a store pay nothing for it.
const concurrentStores = false
