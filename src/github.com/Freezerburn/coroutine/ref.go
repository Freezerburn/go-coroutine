package coroutine

import (
	"log"
)

// Simple reference to a coroutine. Allows external code to send messages to that coroutine, stop it, and check
// various bits of data about it.
type Ref interface {
	Send(v interface{})
	Running() bool
	Name() string
	Id() uint64
	Stop()
}

// Separate struct from the Embeddable coroutine so that the Stop function can behave differently for external code
// versus internal to the coroutine.
type embeddableRef struct {
	e *Embeddable
}

// Puts a message into the mailbox of the coroutine this references.
func (r *embeddableRef) Send(v interface{}) {
	r.e.mailboxLock.Lock()
	r.e.mailbox = append(r.e.mailbox, v)
	r.e.mailboxLock.Unlock()

	// If the coroutine is waiting on the mailbox, let it know. Otherwise continue immediately so the sender
	// doesn't get blocked.
	select {
	case r.e.receiver <- true:
	default:
	}
}

// Whether or not the coroutine this references is still running.
func (r *embeddableRef) Running() bool {
	return r.e.running
}

// The name given to the coroutine this references at start time. If no name was given, a generic name is assigned.
func (r *embeddableRef) Name() string {
	return r.e.name
}

// The unique ID of the coroutine this references.
func (r *embeddableRef) Id() uint64 {
	return r.e.id
}

// Stops the coroutine this references. Will not immediately halt execution of the coroutine, but when it calls any
// of the methods on the Embeddable struct, execution will halt at that point. So if it's in a tight loop, that
// loop will finish.
func (r *embeddableRef) Stop() {
	if !r.e.running {
		log.Printf("Coroutine [%v / %s] attempted to be stopped when it isn't running, possible bug found.",
			r.e.id, r.e.name)
		return
	}

	r.e.running = false
	// If the coroutine is in the middle of attempting to receive something, immediately cause it to stop attempting
	// to receive so it can detect that it needs to stop.
	select {
	case r.e.receiver <- true:
	default:
	}
}
