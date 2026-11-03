package window

// presence remembers whether the window has the focus and whether the
// pointer is over it, so only changes become events. The zero value is a
// focused window with the pointer inside, so a normal start sends nothing.
type presence struct {
	away    bool // not focused
	outside bool // the pointer is not over the window (or it isn't focused)
}

// update reads the state now and gives the events for what changed:
// "focus" when the focus changes, then "leave" or "enter" when the pointer
// leaves or enters the window. The pointer counts as outside while the
// window has no focus, and outside [0, w) x [0, h).
func (p *presence) update(focused bool, x, y, w, h int) []Event {
	var events []Event
	if focused == p.away {
		p.away = !focused
		events = append(events, Event{Type: "focus", Focused: focused})
	}
	outside := !focused || x < 0 || y < 0 || x >= w || y >= h
	if outside != p.outside {
		p.outside = outside
		if outside {
			events = append(events, Event{Type: "leave"})
		} else {
			events = append(events, Event{Type: "enter"})
		}
	}
	return events
}
