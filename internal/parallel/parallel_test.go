package parallel

import (
	"errors"
	"fmt"
	"testing"
)

func TestMapKeepsOrderAndReportsFirstError(t *testing.T) {
	items := make([]int, 500)
	for i := range items {
		items[i] = i
	}
	got, err := Map(items, func(n int) (string, error) { return fmt.Sprint(n * 2), nil })
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range got {
		if v != fmt.Sprint(i*2) {
			t.Fatalf("result %d = %s", i, v)
		}
	}

	_, err = Map(items, func(n int) (int, error) {
		if n == 7 || n == 300 {
			return 0, fmt.Errorf("item %d", n)
		}
		return n, nil
	})
	if err == nil || err.Error() != "item 7" {
		t.Fatalf("expected the lowest-index error, got %v", err)
	}
	if _, err := Map([]int{}, func(int) (int, error) { return 0, errors.New("x") }); err != nil {
		t.Fatal(err)
	}
}
