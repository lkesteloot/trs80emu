// Copyright 2012 Lawrence Kesteloot

package main

// Record a breakpoint at a memory location. If the PC hits this location,
// the machine will stop.
type breakpoint struct {
	pc     uint16
	active bool
}

// Unordered set of breakpoints.
type breakpoints []breakpoint

// Add a breakpoint to a set of breakpoints.
func (bps *breakpoints) add(bp breakpoint) {
	// Here could check that we have a redundant pc.
	*bps = append(*bps, bp)
}

// Returns the active breakpoint at pc or nil if not found.
func (bps breakpoints) find(pc uint16) *breakpoint {
	// Linear is fine, we're not going to have too many of these. If we do, it'd
	// be pretty cheap to have an array of 64k pointer to breakpoints, or even 64k
	// bools (though that may hurt the cache).
	for i := 0; i < len(bps); i++ {
		if bps[i].pc == pc && bps[i].active {
			return &bps[i]
		}
	}

	return nil
}
