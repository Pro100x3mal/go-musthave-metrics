package pool

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestObject - тестовая структура с методом Reset
type TestObject struct {
	Value   int
	Text    string
	Counter int
}

func (t *TestObject) Reset() {
	t.Value = 0
	t.Text = ""
	t.Counter = 0
}

func TestNew(t *testing.T) {
	pool := New(func() *TestObject {
		return &TestObject{}
	})

	require.NotNil(t, pool)
}

func TestPool_GetFromEmpty(t *testing.T) {
	pool := New(func() *TestObject {
		return &TestObject{Value: 42}
	})

	obj := pool.Get()
	require.NotNil(t, obj)
	assert.Equal(t, 42, obj.Value)
}

func TestPool_PutAndGet(t *testing.T) {
	pool := New(func() *TestObject {
		return &TestObject{}
	})

	obj := &TestObject{Value: 100, Text: "test", Counter: 5}

	pool.Put(obj)

	retrieved := pool.Get()
	require.NotNil(t, retrieved)

	assert.Equal(t, 0, retrieved.Value, "Value should be reset to 0")
	assert.Equal(t, "", retrieved.Text, "Text should be reset to empty string")
	assert.Equal(t, 0, retrieved.Counter, "Counter should be reset to 0")
}

func TestPool_MultiplePutAndGet(t *testing.T) {
	pool := New(func() *TestObject {
		return &TestObject{}
	})

	for i := 0; i < 5; i++ {
		obj := &TestObject{Value: i * 10}
		pool.Put(obj)
	}

	for i := 0; i < 5; i++ {
		obj := pool.Get()
		require.NotNil(t, obj)
		assert.Equal(t, 0, obj.Value)
	}
}

func TestPool_ConcurrentAccess(t *testing.T) {
	pool := New(func() *TestObject {
		return &TestObject{}
	})

	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				obj := pool.Get()
				obj.Value = id
				obj.Counter = j
				pool.Put(obj)
			}
		}(i)
	}

	wg.Wait()

	obj := pool.Get()
	assert.Equal(t, 0, obj.Value)
	assert.Equal(t, 0, obj.Counter)
}
