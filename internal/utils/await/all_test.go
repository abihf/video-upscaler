package await

import (
	"errors"
	"sync/atomic"
	"testing"
)

func TestAllReturnsNilWhenAllCallsSucceed(t *testing.T) {
	var calls atomic.Int32

	err := All(func(value int) error {
		calls.Add(1)
		if value < 0 {
			t.Fatalf("unexpected value: %d", value)
		}
		return nil
	}, 1, 2, 3)

	if err != nil {
		t.Fatalf("All returned error: %v", err)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("All called function %d times, want 3", got)
	}
}

func TestAllReturnsFirstObservedError(t *testing.T) {
	wantErr := errors.New("boom")

	err := All(func(value int) error {
		if value == 2 {
			return wantErr
		}
		return nil
	}, 1, 2, 3)

	if !errors.Is(err, wantErr) {
		t.Fatalf("All returned %v, want %v", err, wantErr)
	}
}
