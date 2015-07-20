Generic Resource Pool
======================

Manage resources such as database connections etc. with
this package.

Pool can be configured by using the options struct while
initializing it. The pre and post hooks can be used to
test resource. The eviction policy can be set by 
the resource as well. 

The pool will test for freshness on a schedule set 
by the client. 

Installation:
---

`go get github.com/avarghes1/go_pool/pool`

Import:
---

`import github.com/avarghes1/go_pool/pool`

Usage
---

```
	r := new(Testresource)
	p, _ := pool.Initialize(r, pool.Options{PoolSize: 10,
		Timeout:           time.Second,
		EvictionTest:      true,
		EvictTestSchedule: time.Second * 1})
	for i := 0; i < 10; i++ {
		a, err := p.Acquire()
		fmt.Println("%+v %+v", a, err)
		p.Release(a)
	}
```
