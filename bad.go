package eventdb

import (
	"regexp"
)

// IsBadEvent applies some heuristics to remove spammy events or expensive ones
// that aren't practical to show up at without previous notice.
//
// Not sure if I want to keep this since it makes things less random. Perhaps
// there's some machine learning magic I can do to filter events while
// minimizing bias?
func IsBadEvent(event Event) bool {
	for _, filt := range nameFilters {
		if filt.MatchString(event.Name) {
			return true
		}
	}
	for _, filt := range descFilters {
		if filt.MatchString(event.Description) {
			return true
		}
	}

	return false
}

var nameFilters = []*regexp.Regexp{
	// If it's sold out or canceled you'll be turned away.
	regexp.MustCompile(`(?i)\bSold Out\b`),
	regexp.MustCompile(`(?i)\bCancel\b`),
	regexp.MustCompile(`(?i)\bgeschlossene\b`), // German
	regexp.MustCompile(`(?i)\babgesagte\b`),    // German
	regexp.MustCompile(`(?i)\bannulliert\b`),   // German

	// Don't go to Facebook funerals.
	regexp.MustCompile(`(?i)\bFuneral\b`),

	// I have nothing against bars, but too many bars seem to be using Facebook
	// events as a marketing channel. FB is flooded with "tap takeovers" and other
	// beer sales events. I've been to a ton of these events and they're usually
	// expensive and terrible.
	regexp.MustCompile(`(?i)\bbar\b`),
	regexp.MustCompile(`(?i)\bpub\b`),
}

var descFilters = []*regexp.Regexp{
	// Facebook events should be free.
	//
	// At some point it might be nice to add some price parsing and allow people
	// to filter by price range. I'd be willing to spend $5 on most events, but
	// $50 is too much especially if you're going to more than one in a night.
	regexp.MustCompile(`(\$|¥|₹|₡|₱|£|€|₩|₨|﷼|₱|₽)`),
	regexp.MustCompile(`(?i)dollars`),
	regexp.MustCompile(`Rs *\d`), // India

	// It's a bad idea to send people to support groups. I know this from
	// experience. It can be intrusive to show up at a support event for a group
	// you're not a part of.
	//
	// Of course, this filters out events for groups that you _are_ a part of, and
	// groups that are supporting one group want diverse participation, which is
	// a shame. Maybe we can be smarter about this filter later.
	regexp.MustCompile(`(?i)support group`),
	regexp.MustCompile(`(?i)(men|women|children) only`),

	// Right now we're only generating events happening in the next few hours.
	// If an RSVP is required then you might be turned away.
	regexp.MustCompile(`(?i)regist`),
	regexp.MustCompile(`(?i)RSVP`),
	regexp.MustCompile(`(?i)anmelden`),  // German
	regexp.MustCompile(`(?i)anmeldung`), // German
}
