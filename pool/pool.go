// Generic Pool.
//
// Manage resources such as database connections etc. with
// this package.
//
// Example:
//	r := new(Testresource)
//	p, _ := pool.Initialize(r, pool.Options{PoolSize: 10,
//		Timeout:           time.Second,
//		EvictionTest:      true,
//		EvictTestSchedule: time.Second * 1})
//	for i := 0; i < 10; i++ {
//		a, err := p.Acquire()
//		fmt.Println("%+v %+v", a, err)
//		p.Release(a)
//	}
//
// @author: avarghese
package pool

import (
	"errors"
	"sync"
	"time"
)

type (
	// Following is a bad example of creating a resource
	//
	// Example:
	//        type (
	//               Testresource struct{}
	//        )
	//        func (t *Testresource) Ping() bool {
	//                return true
	//        }
	//        func (t *Testresource) Evict() bool {
	//                println("Evict")
	//                return true
	//        }
	//        func (t *Testresource) PreAcquire() error {
	//                println("pre borrow")
	//                if ok := t.Ping(); !ok {
	//                        n, err := t.Add()
	//                        if err != nil {
	//                                return err
	//                        }
	//                        t = n.(*Testresource)
	//                }
	//                return nil
	//        }
	//        func (t *testresource) PostAcquire() error {
	//                println("post borrow")
	//                return nil
	//        }
	//        func (t *testresource) PreRelease() error {
	//                println("pre return")
	//                if ok := t.Ping(); !ok {
	//                        n, err := t.Add()
	//                        if err != nil {
	//                                return err
	//                        }
	//                        t = n.(*Testresource)
	//                }
	//                return nil
	//        }
	//        func (t *Testresource) PostRelease() error {
	//                println("post return")
	//                return nil
	//        }
	//        func (t *Testresource) Add() (pool.Resource, error) {
	//                return t, nil
	//        }
	//
	Resource interface {
		Add() (Resource, error) // Create a resource
		Ping() bool             // Check if resource is still valid
		Evict() bool            // Evict a resource
		PreAcquire() error      // Process Resource Before Acquire
		PostAcquire() error     // Process Resource After Acquire
		PreRelease() error      // Process Resource Before Release
		PostRelease() error     // Process Resource After Release
	}
	Options struct {
		PoolSize          int64         // The number of resources in the pool
		Timeout           time.Duration // Timeout for acquiring a resource
		EvictionTest      bool          // Refresh the pool?
		EvictTestSchedule time.Duration // Schedule for testing resources
	}
	Pool struct {
		c chan Resource // Channel for Resources
		n int64         // number of resources in pool
		l sync.Mutex    //Mutex
		o Options       // pool options
	}
)

// Internal function for testing/refreshing resources.
func (p *Pool) refreshPool() {
	p.l.Lock()
	defer p.l.Unlock()
	for i := int64(0); i < p.n; i++ {
		select {
		case r := <-p.c:
			if r.Evict() {
				t, err := r.Add()
				if err != nil {
					break
				}
				r = t
			}
			p.c <- r
		case <-time.After(p.o.Timeout):
			continue
		}
	}
}

// Initialize a pool
//
// Usage:
//
//      r := new(Testresource)
//	p, _ := pool.Initialize(r, pool.Options{PoolSize: 10,
//		Timeout:           time.Second,
//		EvictionTest:      true,
//		EvictTestSchedule: time.Second * 1}
//      )
//
func Initialize(r Resource, o Options) (*Pool, error) {
	p := new(Pool)
	p.c = make(chan Resource, o.PoolSize)
	for i := int64(0); i < o.PoolSize; i++ {
		r, err := r.Add()
		if err != nil {
			return nil, err
		}
		p.c <- r
	}
	p.n = o.PoolSize
	p.o = o
	// If pool needs to be tested, schedule the refresh
	if o.EvictionTest {
		tick := time.NewTicker(o.EvictTestSchedule)
		go func() {
			for _ = range tick.C {
				p.refreshPool()
			}
		}()
	}
	return p, nil
}

// Acquire a resource from the pool.
// Will time out if option is set.
func (p *Pool) Acquire() (r Resource, err error) {
	select {
	case r = <-p.c:
		if err = r.PreAcquire(); err != nil {
			return nil, err
		}
		p.l.Lock()
		p.n--
		p.l.Unlock()
		if err = r.PreAcquire(); err != nil {
			return nil, err
		}
		return r, err
	case <-time.After(p.o.Timeout):
		return nil, errors.New("Timeout")
	}
}

// Release a resource back to the pool
func (p *Pool) Release(r Resource) (err error) {
	if err := r.PreRelease(); err != nil {
		return err
	}
	p.c <- r
	p.l.Lock()
	p.n++
	p.l.Unlock()
	if err := r.PostRelease(); err != nil {
		return err
	}
	return err
}
