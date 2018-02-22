package coroutine

import (
	"time"
	"sync"
	"log"
)

// The base struct that has all the functions necessary to operate on a coroutine. It is designed to be used in one of
// two ways:
// - As the argument to a func passed to one of the Start functions. It can then be used as a proxy for the
//   coroutine function. e.g.: `msg := e.Recv()`
// - Embedded directly into another struct, so that the struct can call the coroutine functions as if they were its
//   own. e.g.: `type A struct { coroutine.Embeddable }` The struct that embeds the coroutine struct can then
//   implement the Start method on the Starter interface and an instance of it can be passed directly to one of the
//   Start functions. Embeddable MUST be embedded as a non-pointer, and the struct embedding it MUST be used as a
//   pointer.
type Embeddable struct {
	id           uint64
	name         string
	waitTimer    *time.Timer
	receiver     chan bool
	receiveTimer *time.Timer
	mailbox      []interface{}
	mailboxLock  sync.Mutex
	running      bool
}

// Pauses execution of this coroutine for the given duration to allow other coroutines to run.
//
// If this coroutine has been stopped by external code using the Ref returned by all Start functions, then it will
// immediately stop, and no further code outside of deferred functions will be executed in this coroutine.
func (e *Embeddable) Pause(duration time.Duration) {
	if !e.running {
		// Every coroutine is wrapped in a function that recovers from a panic, so this is guaranteed to immediately
		// stop execution of the coroutine completely without stopping the rest of the program.
		panic(Stop{})
	}

	// Guarantee the timer channel is drained.
	// Despite the docs saying to only check the return value of Stop, it's possible for Stop to return false without
	// there being an item in the channel, so we also double-check if there is an item in the channel before attempting
	// to drain it.
	if !e.waitTimer.Stop() && len(e.waitTimer.C) > 0 {
		<-e.waitTimer.C
	}
	e.waitTimer.Reset(duration)
	<-e.waitTimer.C

	// Since there's a period of time that this is doing nothing, there's a chance that external code could stop
	// this coroutine while it's paused. So we check that before returning control to the coroutine.
	if !e.running {
		panic(Stop{})
	}
}

// Checks the mailbox for any sent messages. If none are in the mailbox, this function will halt the coroutine until
// one gets sent.
//
// If this coroutine has been stopped by external code using the Ref returned by all Start functions, then it will
// immediately stop, and no further code outside of deferred functions will be executed in this coroutine.
func (e *Embeddable) Recv() interface{} {
	if !e.running {
		panic(Stop{})
	}

	e.mailboxLock.Lock()
	if len(e.mailbox) == 0 {
		e.mailboxLock.Unlock()
		<-e.receiver

		if !e.running {
			panic(Stop{})
		}

		e.mailboxLock.Lock()
	}

	r := e.mailbox[0]
	e.mailbox = e.mailbox[1:]
	e.mailboxLock.Unlock()
	return r
}

// Checks if the mailbox contains anything. If it does, that value and true are returned. If it doesn't, the
// coroutine will pause for up to duration time. If a value is put into the mailbox within that time, that value and
// true are returned. If nothing was put into the mailbox during that time, nil and false are returned.
//
// If this coroutine has been stopped by external code using the Ref returned by all Start functions, then it will
// immediately stop, and no further code outside of deferred functions will be executed in this coroutine.
func (e *Embeddable) RecvFor(duration time.Duration) (interface{}, bool) {
	if !e.running {
		panic(Stop{})
	}

	if duration <= 0 {
		return e.RecvImmediate()
	}

	e.mailboxLock.Lock()
	if len(e.mailbox) == 0 {
		e.mailboxLock.Unlock()

		if !e.receiveTimer.Stop() && len(e.receiveTimer.C) > 0 {
			<-e.receiveTimer.C
		}

		e.receiveTimer.Reset(duration)
		select {
		case <-e.receiver:
		case <-e.receiveTimer.C:
		}

		if !e.running {
			panic(Stop{})
		}

		e.mailboxLock.Lock()
	}

	if len(e.mailbox) == 0 {
		e.mailboxLock.Unlock()
		return nil, false
	}

	r := e.mailbox[0]
	e.mailbox = e.mailbox[1:]
	e.mailboxLock.Unlock()
	return r, true
}

// Checks if the mailbox contains anything. If it doesn't, nil and false are returned. If something is in the mailbox,
// that value and true are returned. The found value is removed from the mailbox.
//
// If this coroutine has been stopped by external code using the Ref returned by all Start functions, then it will
// immediately stop, and no further code outside of deferred functions will be executed in this coroutine.
func (e *Embeddable) RecvImmediate() (interface{}, bool) {
	if !e.running {
		panic(Stop{})
	}

	e.mailboxLock.Lock()
	if len(e.mailbox) == 0 {
		e.mailboxLock.Unlock()
		return nil, false
	}

	r := e.mailbox[0]
	e.mailbox = e.mailbox[1:]
	e.mailboxLock.Unlock()
	return r, true
}

// Immediately stop this coroutine. No more code in the coroutine will run, so be sure to do any cleanup work before
// calling this function, or have a deferred function that will do your cleanup work.
func (e *Embeddable) Stop() {
	if !e.running {
		log.Printf("Coroutine [%v / %s] attempted to stop itself when it isn't running, possible bug found.",
			e.id, e.name)
	}
	e.running = false
	panic(Stop{})
}

