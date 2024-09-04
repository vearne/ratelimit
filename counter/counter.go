package counter

import "sync"

type Counter struct {
	sync.Mutex
	x int
}

func NewCounter() *Counter {
	c := Counter{x: 0}
	return &c
}

func (c *Counter) Incr() int {
	c.Lock()
	defer c.Unlock()
	c.x++
	return c.x
}
