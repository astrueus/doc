package errs

import (
	"errors"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	e := New(404, "not found")
	if e.Code != 404 || e.Msg != "not found" || e.Cause != nil {
		t.Fatalf("New: got %+v", e)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("db down")
	e := Wrap(500, "internal", cause)
	if e.Code != 500 || e.Msg != "internal" || e.Cause != cause {
		t.Fatalf("Wrap: got %+v", e)
	}
}

func TestBizErrorError(t *testing.T) {
	e := New(1, "msg")
	if e.Error() != "msg" {
		t.Fatalf("Error: got %q want %q", e.Error(), "msg")
	}
	var nilErr *BizError
	if nilErr.Error() != "" {
		t.Fatal("nil BizError Error should return empty")
	}
}

func TestBizErrorUnwrap(t *testing.T) {
	cause := errors.New("root")
	e := Wrap(1, "wrap", cause)
	if e.Unwrap() != cause {
		t.Fatalf("Unwrap: got %v want %v", e.Unwrap(), cause)
	}
	var nilErr *BizError
	if nilErr.Unwrap() != nil {
		t.Fatal("nil BizError Unwrap should return nil")
	}
}

func TestAsBiz(t *testing.T) {
	inner := New(400, "bad request")
	wrapped := fmt.Errorf("outer: %w", inner)
	got, ok := AsBiz(wrapped)
	if !ok || got != inner {
		t.Fatalf("AsBiz wrapped: got (%v, %v)", got, ok)
	}
	_, ok = AsBiz(errors.New("plain"))
	if ok {
		t.Fatal("AsBiz plain error should be false")
	}
}
