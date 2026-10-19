// Package jsengine runs bundled JavaScript libraries, such as Pele and Temml,
// inside the binary at build time. It uses goja, a JavaScript engine written
// in Go, so Nida stays a single static binary.
package jsengine

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// timeout stops a call that would otherwise hang the build.
var timeout = 10 * time.Second

// ErrTimeout is returned when a call runs longer than the timeout.
var ErrTimeout = errors.New("took longer than 10 seconds")

// Pool is a small set of runtimes that have loaded one library. Runtimes are
// created on first use, so a site that never calls the library never loads
// it. A runtime is not safe for concurrent use, so each call takes one from
// the pool.
type Pool struct {
	name   string
	source []byte
	entry  string

	once    sync.Once
	program *goja.Program
	err     error

	idle  chan *VM
	slots chan struct{}
}

// VM is a runtime with the library loaded.
type VM struct {
	*goja.Runtime
	// Entry is the library function named when the pool was created.
	Entry goja.Callable
}

// NewPool returns a pool for a library. source is the gzipped script, and
// entry is the function calls use, such as "pele.render".
func NewPool(name string, source []byte, entry string) *Pool {
	size := min(runtime.GOMAXPROCS(0), 4)
	return &Pool{
		name:   name,
		source: source,
		entry:  entry,
		idle:   make(chan *VM, size),
		slots:  make(chan struct{}, size),
	}
}

// Run calls fn with a runtime from the pool. A call that runs longer than
// 10 seconds is interrupted and returns ErrTimeout.
func (p *Pool) Run(fn func(vm *VM) error) error {
	vm, err := p.acquire()
	if err != nil {
		return err
	}
	defer func() { p.idle <- vm }()

	timer := time.AfterFunc(timeout, func() { vm.Interrupt("timeout") })
	err = fn(vm)
	timer.Stop()
	vm.ClearInterrupt()

	var interrupted *goja.InterruptedError
	if errors.As(err, &interrupted) {
		return ErrTimeout
	}
	return err
}

func (p *Pool) acquire() (*VM, error) {
	select {
	case vm := <-p.idle:
		return vm, nil
	default:
	}
	select {
	case vm := <-p.idle:
		return vm, nil
	case p.slots <- struct{}{}:
		vm, err := p.newVM()
		if err != nil {
			<-p.slots
			return nil, err
		}
		return vm, nil
	}
}

func (p *Pool) newVM() (*VM, error) {
	p.once.Do(func() {
		src, err := gunzip(p.source)
		if err == nil {
			p.program, err = goja.Compile(p.name, string(src), true)
		}
		if err != nil {
			p.err = fmt.Errorf("load %s: %w", p.name, err)
		}
	})
	if p.err != nil {
		return nil, p.err
	}
	rt := goja.New()
	if _, err := rt.RunProgram(p.program); err != nil {
		return nil, fmt.Errorf("load %s: %w", p.name, err)
	}
	var value goja.Value = rt.GlobalObject()
	for _, part := range strings.Split(p.entry, ".") {
		obj, ok := value.(*goja.Object)
		if !ok {
			return nil, fmt.Errorf("load %s: %s not found", p.name, p.entry)
		}
		value = obj.Get(part)
	}
	entry, ok := goja.AssertFunction(value)
	if !ok {
		return nil, fmt.Errorf("load %s: %s is not a function", p.name, p.entry)
	}
	return &VM{Runtime: rt, Entry: entry}, nil
}

// Thrown returns the JavaScript value a call threw, or nil if err is not a
// JavaScript exception.
func Thrown(err error) *goja.Object {
	var exception *goja.Exception
	if !errors.As(err, &exception) {
		return nil
	}
	obj, _ := exception.Value().(*goja.Object)
	return obj
}

// String returns a property of a thrown value as a string, or "".
func String(obj *goja.Object, name string) string {
	if v := obj.Get(name); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		return v.String()
	}
	return ""
}

// Int returns a property of a thrown value as an integer, or 0 when it is
// missing or outside the int32 range, which no line or column number needs.
func Int(obj *goja.Object, name string) int {
	if v := obj.Get(name); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		if n := v.ToInteger(); n >= math.MinInt32 && n <= math.MaxInt32 {
			return int(n)
		}
	}
	return 0
}

// Cache keeps rendered output by key. serve keeps it across rebuilds, so
// it is cleared when full rather than growing without bound.
type Cache struct {
	mu sync.Mutex
	m  map[string]string
}

const cacheLimit = 4096

func (c *Cache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[key]
	return v, ok
}

func (c *Cache) Put(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil || len(c.m) >= cacheLimit {
		c.m = map[string]string{}
	}
	c.m[key] = value
}

func gunzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}
