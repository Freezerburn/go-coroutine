A simple coroutine library for Golang.
======================================

While it's very easy to start goroutines in Go, there is extra work that needs to be done manually if you want to
communicate with that goroutine or stop it easily. This is a very simple wrapper around various bits of Go's
standard library to manage these tasks for you in as easy to use a way as possible.

# Example

    type MyCoroutine struct {
        lines int
        coroutine.Embeddable
    }

    func (c *MyCoroutine) Start() {
        for {
            switch msg := c.Recv().(type) {
            case int:
                fmt.Printf("Message %v: Received integer [%v]\n", c.line, msg)
            case string:
                fmt.Printf("Message %v: Received string [%s]\n", c.line, msg)
            }
            c.line++
        }

        fmt.Println("This never prints")
    }

    func main() {
        r := coroutine.Start(&MyCoroutine{})
        for i := 0; i < 100; i++ {
            r.Send(i);
            r.Send("msg " + strconv.Itoa(i))
        }
        time.Sleep(5 * time.Millisecond)
        r.Stop()
        time.Sleep(5 * time.Millisecond)
        fmt.Println("All done. All messages should have been printed, but not the println at the end of Start.")
    }

### Documentation

## Base functions

The `Function` type referenced by some Start function variants is: `type Function func(embeddable *Embeddable)`

* `func StartFunc(f Function) Ref`: Starts a coroutine with a default name by using the given function.
* `func StartFuncName(name string, f Function) Ref`: Starts a coroutine with the given name by using the given function.
* `func Start(s Starter) Ref`: Starts a coroutine with a default name using the struct implementing the Starter
interface. Usually the struct will embed the Embeddable struct as a value.
* `func StartName(s Starter) Ref`: Starts a coroutine with the given name using the struct implementing the Starter
interface. Usually the struct will embed the Embeddable struct as a value.

# Embeddable

Designed to be embedded into a struct as a value as shown in the above example. If one of the Start methods that take
a function is used, a pointer to one of these will be passed in so you don't have to always create a struct that
implements the `Start`function of the `Starter` interface.


All Recv function variants get messages in First-In First-Out order.

Functions available:

* `func Recv() interface{}`: Waits until a message arrives in the coroutine's mailbox.
* `func RecvFor(duration time.Duration) (interface{}, bool)`: Waits the specified amount of time for a message, and
returns false if a message did not arrive within that given period of time. If the duration is <= 0, acts the same as
RecvImmediate.
* `func RecvImmediate() (interface{}, bool)`: If no messages are in the mailbox, it will return false.
* `func Pause(duration time.Duration)`: Pauses the coroutine for the given amount of time. This is useful as opposed
to `time.Sleep` because if the coroutine is `Stop`ped via the Ref returned from a Start function, the coroutine will
not have any further code run except for deferred functions.
* `func Stop()`: Immediately stops the coroutine and all code running in it. Only deferred functions will run when
this is used. Might be useful as opposed to a simple `return` if you are deep in a call stack.


# Ref

Functions available:

* `Send(v interface{})`: Send a message to the referenced coroutine.
* `Running() bool`: Whether or not the referenced coroutine is still running.
* `Name() string`: The name of the referenced coroutine.
* `Id() uint64`: The unique ID of the referenced coroutine.
* `Stop()`: Stop the referenced coroutine. Code in the coroutine will only stop running when it calls one of the
functions from the Embeddable struct. So if it is in the middle of handling a message or something, it will finish
what it is doing.
