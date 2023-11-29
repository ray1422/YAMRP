package async

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAsync(t *testing.T) {
	f := Async(func() int {
		return 1
	})
	assert.Equal(t, 1, Await(f))
}

func TestAwaitAny(t *testing.T) {
	f1 := Async(func() int {
		// sleep for 1 second
		time.Sleep(1 * time.Second)
		return 1
	})
	f2 := Async(func() int {
		// sleep for 2 seconds
		time.Sleep(2 * time.Second)
		return 2
	})
	assert.Equal(t, 1, AwaitAny(f1, f2))
}

func TestAwaitUnordered(t *testing.T) {
	f1 := Async(func() int {
		// sleep for 1 second
		time.Sleep(1 * time.Second)
		return 1
	})
	f2 := Async(func() int {
		// sleep for 2 seconds
		time.Sleep(2 * time.Second)
		return 2
	})
	time.Sleep(3 * time.Second)
	ch := AwaitUnordered(f1, f2)
	ret := []int{}

	for v, ok := <-ch; ok; v, ok = <-ch {
		ret = append(ret, v)
	}

	assert.Contains(t, ret, 1)
	assert.Contains(t, ret, 2)
}
