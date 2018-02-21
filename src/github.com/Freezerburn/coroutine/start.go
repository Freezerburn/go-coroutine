package coroutine

import (
	"sync"
	"time"
)

// The type that is passed to panic whenever a coroutine is stopping itself. If a coroutine function has a
// deferred function which calls recover, this should be checked for to be ignored. If the deferred functions which
// recovers would cause a panic, and this was the value that came from the return value of recover, this should be
// what is given to panic to continue normal execution of the program.
type Stop struct {
}

func (s *Stop) Error() string {
	return "coroutine stop"
}

// Signature of the func that can be started as a coroutine. Receives the embeddable struct so that the func can
// call methods on it to act as a coroutine.
type Function func(embeddable *Embeddable)
type Starter interface {
	Start()
	Embedded() *Embeddable
}

const (
	defaultName = "Default Coroutine Name"
)

var (
	nextId     uint64 = 1
	nextIdLock sync.Mutex
)

func StartFunc(f Function) Ref {
	return StartFuncName(defaultName, f)
}

func StartFuncName(name string, f Function) Ref {
	next := &Embeddable{
		name:         name,
		waitTimer:    time.NewTimer(0),
		receiver:     make(chan bool),
		receiveTimer: time.NewTimer(0),
		running:      true,
	}

	nextIdLock.Lock()
	next.id = nextId
	nextId++
	nextIdLock.Unlock()

	go func() {
		defer func() {
			// Ensure external code will know that this coroutine is stopped if the program doesn't end due to the
			// panic.
			next.running = false
			// Close down all the coroutine's resources.
			next.waitTimer.Stop()
			next.receiveTimer.Stop()
			close(next.receiver)

			if e := recover(); e != nil {
				if _, ok := e.(Stop); ok {
					// Stop requested for this coroutine, so we just let the goroutine end.
				} else {
					// Repanic since it came from code that isn't part of the coroutine library.
					panic(e)
				}
			}
		}()

		f(next)
	}()

	return &embeddableRef{next}
}

func Start(s Starter) Ref {
	return StartName(defaultName, s)
}

func (e *Embeddable) Embedded() *Embeddable {
	return e
}

func StartName(name string, s Starter) Ref {
	e := s.Embedded()
	e.name = name
	e.waitTimer = time.NewTimer(0)
	e.receiver = make(chan bool)
	e.receiveTimer = time.NewTimer(0)
	e.running = true

	nextIdLock.Lock()
	e.id = nextId
	nextId++
	nextIdLock.Unlock()

	go func() {
		defer func() {
			// Ensure external code will know that this coroutine is stopped if the program doesn't end due to the
			// panic.
			e.running = false
			// Close down all the coroutine's resources.
			e.waitTimer.Stop()
			e.receiveTimer.Stop()
			close(e.receiver)

			if e := recover(); e != nil {
				if _, ok := e.(Stop); ok {
					// Stop requested for this coroutine, so we just let the goroutine end.
				} else {
					// Repanic since it came from code that isn't part of the coroutine library.
					panic(e)
				}
			}
		}()

		s.Start()
	}()

	return &embeddableRef{e}
}
