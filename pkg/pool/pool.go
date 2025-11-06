package pool

import "sync"

type Resetter interface {
	Reset()
}

type Pool[T Resetter] struct {
	pool    sync.Pool
	newFunc func() T
}

func New[T Resetter](newFunc func() T) *Pool[T] {
	return &Pool[T]{
		newFunc: newFunc,
		pool: sync.Pool{
			New: func() interface{} {
				return newFunc()
			},
		},
	}
}

func (p *Pool[T]) Get() T {
	obj := p.pool.Get()
	if obj == nil {
		return p.newFunc()
	}
	return obj.(T)
}

func (p *Pool[T]) Put(obj T) {
	obj.Reset()
	p.pool.Put(obj)
}
