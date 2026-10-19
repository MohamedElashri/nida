package jsengine

import (
	"bytes"
	"compress/gzip"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja"
)

func gz(t *testing.T, src string) []byte {
	t.Helper()
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write([]byte(src)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

const lib = `var lib = {
  echo: function (s) { return "<" + s + ">"; },
  fail: function () { var e = new Error("bad input"); e.name = "LibError"; e.line = 3; throw e; },
  spin: function () { for (;;) {} }
};`

func TestPoolCallsEntryAndReportsThrownValues(t *testing.T) {
	pool := NewPool("lib", gz(t, lib), "lib.echo")
	var got string
	err := pool.Run(func(vm *VM) error {
		v, err := vm.Entry(goja.Undefined(), vm.ToValue("x"))
		if err == nil {
			got = v.String()
		}
		return err
	})
	if err != nil || got != "<x>" {
		t.Fatalf("got %q, %v", got, err)
	}

	err = NewPool("lib", gz(t, lib), "lib.fail").Run(func(vm *VM) error {
		_, err := vm.Entry(goja.Undefined())
		return err
	})
	thrown := Thrown(err)
	if thrown == nil || String(thrown, "name") != "LibError" || Int(thrown, "line") != 3 || String(thrown, "missing") != "" {
		t.Fatalf("unexpected thrown value from %v", err)
	}
}

func TestPoolRejectsMissingEntry(t *testing.T) {
	err := NewPool("lib", gz(t, lib), "lib.nope").Run(func(*VM) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "lib.nope") {
		t.Fatalf("expected a missing-entry error, got %v", err)
	}
}

func TestPoolInterruptsLongCalls(t *testing.T) {
	defer func(old time.Duration) { timeout = old }(timeout)
	timeout = 50 * time.Millisecond
	pool := NewPool("lib", gz(t, lib), "lib.spin")
	err := pool.Run(func(vm *VM) error {
		_, err := vm.Entry(goja.Undefined())
		return err
	})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected a timeout, got %v", err)
	}
	// The runtime goes back to the pool and must work again.
	if err := pool.Run(func(vm *VM) error { _, err := vm.RunString("1"); return err }); err != nil {
		t.Fatalf("runtime unusable after timeout: %v", err)
	}
}
