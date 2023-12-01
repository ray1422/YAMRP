package async

// Future for async
// cation: Future must be initialized with `NewFuture` or `Async`
//
// usage:
//
// // define a function
//
//	func f() Future[int] {
//		return Ret(1)
//	}
//
// // call the function
//
// // get the result
//
//	res := Await(Async(f))
type Future[T any] struct {
	ch chan T
}

// Ch exposes the channel of a Future for Select.
func (f Future[T]) Ch() <-chan T {
	return f.ch
}

func (f Future[T]) Await() T {
	return <-f.ch
}

// Resolve resolves a Future with a value
func (f *Future[T]) Resolve(v T) {
	f.ch <- v
}

// NewFuture creates a new Future with buffered channel
func NewFuture[T any]() Future[T] {
	return Future[T]{ch: make(chan T, 1)}
}

// Async runs a function asynchronously
// usage:
//
//	f := Async(func() int {
//		return 1
//	})
//	fmt.Println(Await(f))
func Async[T any](f func() T) Future[T] {
	ret := Future[T]{ch: make(chan T, 1)}
	go func() {
		ret.ch <- f()
	}()
	return ret
}

// Await waits for the result of a Future
func Await[T any](f Future[T]) T {
	return <-f.ch
}

// Ret returns a Future with a value
// usage:
//
//	return Ret(1)
func Ret[T any](v T) Future[T] {
	ret := Future[T]{ch: make(chan T, 1)}
	go func() {
		ret.ch <- v
	}()
	return ret
}

// AwaitAll waits for the results of multiple Futures
func AwaitAll[T any](fs ...Future[T]) []T {
	ret := make([]T, len(fs))
	for i, f := range fs {
		ret[i] = Await(f)
	}
	return ret
}

// AwaitAny waits for the result of any Future
func AwaitAny[T any](fs ...Future[T]) T {
	for {
		for _, f := range fs {
			select {
			case ret := <-f.ch:
				return ret
			default:
			}
		}
	}
}

// AwaitUnordered waits for the results of multiple Futures in any order
func AwaitUnordered[T any](fs ...Future[T]) chan T {
	ret := make(chan T, len(fs))
	cnt := 0
	for cnt < len(fs) {
		for _, f := range fs {
			select {
			case ret <- <-f.ch:
				cnt++
				break
			default:
			}
		}
	}
	close(ret)
	return ret
}
