package main

type breakpoint struct {
	pc     word
	active bool
}

type breakpoints []breakpoint

func (bps *breakpoints) add(bp breakpoint) {
	// Here could check that we have a redundant pc.
	*bps = append(*bps, bp)
}

// Returns the active breakpoint at pc or nil if not found.
func (bps breakpoints) find(pc word) *breakpoint {
	// Linear is fine, we're not going to have too many of these. If we do, it'd
	// be pretty cheap to have an array of 64k pointer to breakpoints, or even 64k
	// bools.
	for i := 0; i < len(bps); i++ {
		if bps[i].pc == pc && bps[i].active {
			return &bps[i]
		}
	}

	return nil
}
