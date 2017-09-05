package errbag

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type myError struct{ error }

type myErrorBag struct {
	ErrorBag
}

func (meb *myErrorBag) WrapError(err error) error {
	if merr, ok := err.(*myError); ok {
		return merr
	}
	return &myError{err}
}

type otherThing struct {
	*myErrorBag
}

func logf(msg string, a ...interface{}) {
	if logfunc == nil {
		return
	}

	logfunc(msg, a...)
}

var logfunc func(msg string, a ...interface{})

func TestErrorBag(t *testing.T) {
	logfunc = t.Logf
	defer func() { logfunc = nil }()

	mb := &myErrorBag{}
	mb.ErrorWrapper(mb)

	if got := mb.ErrorOrNil(); !errorBagsEqual(got, nil) {
		t.Errorf("initial value: Got %v; Wanted %v", got, nil)
	}

	if got := mb.Errorf("first error"); !errorBagsEqual(got, mb) {
		t.Errorf("adding first error: Got %#v; Wanted %#v", got, mb)
	}

	ot := &otherThing{mb}

	if got := ot.Errorf("second error"); !errorBagsEqual(got, mb) {
		t.Errorf("adding second error: Got %#v; Wanted %#v", got, mb)
	}

	want := &ErrorBag{
		errs: []error{
			errors.New("first error"),
			errors.New("second error"),
		},
	}

	if !errorBagsEqual(mb, want) {
		t.Errorf("after second error: Got %#v; Wanted %#v", mb, want)
	}

	// Here we grab the base ErrorBag that's embedding in `mb` (which is embedded
	// in `ot`) and then  try to `Add()` it using `ot`. Since we have not updated
	// `want` we intend for this to *not* get added (since `Add()` is supposed to
	// avoid adding itself).
	if got := ot.Add(&mb.ErrorBag); !errorBagsEqual(got, want) {
		t.Errorf("ot.Add(mb.ErrorBag) != want\n   Got(%#v);\nWanted(%#v)", got, want)
	}

	we := errors.New("a wrapped error")
	want.errs = append(want.errs, &myError{we})

	if got := ot.Wrap(we); !errorBagsEqual(got, want) {
		t.Errorf("ot.Wrap(%#v) != want\n   Got(%#v);\nWanted(%#v)", we, got, want)
	}

	/*
		Visit(err, func(i int, e error) {
			t.Logf("  2.%d: err: %v [%T]", i, e, e)
		})

		err3 := mb.Wrap(err2)
		t.Logf("3: err: %v [%T]", err3, err3)

		err4 := mb.Wrap(errors.New("this is a new error"))
		t.Logf("4: err: %v [%T]", err4, err4)

		Visit(err, func(i int, e error) {
			t.Logf("  4.%d: err: %v [%T]", i, e, e)
		})
	*/
}

func errorBagsEqual(a, b error) bool {
	if a == b {
		logf("a and b are the same object ==> TRUE")
		return true
	}

	if a == nil && b == nil {
		logf("both are nil ==> TRUE")
		return true
	}

	if a == nil || b == nil {
		logf("one is nil: a:%v b:%v ==> FALSE", a == nil, b == nil)
		return false
	}

	// neither is nil

	eba := AsErrorBag(a)
	if eba == nil {
		logf("a is not *ErrorBag")
		return false
	}

	ebb := AsErrorBag(b)
	if ebb == nil {
		logf("b is not *ErrorBag")
		return false
	}

	if len(eba.errs) != len(ebb.errs) {
		logf("different lengths: %d != %d ==> FALSE", len(eba.errs), len(ebb.errs))
		return false
	}

	for i, erra := range eba.errs {
		errb := ebb.errs[i]

		if erra == nil && errb == nil {
			logf("a.errs[%d] == b.errs[%d] == nil ==> CONT", i, i)
			continue
		}

		if erra == nil || errb == nil {
			logf("one of errs[%d] is nil; a:%v b:%v ==> FALSE", i, erra == nil, errb == nil)
			return false
		}

		if reflect.TypeOf(erra) != reflect.TypeOf(errb) {
			logf("types of errs[%d] differ; a:%T b:%T ==> FALSE", i, erra, errb)
			return false
		}

		if erra.Error() != errb.Error() {
			logf("strings of errs[%d] differ; a:%q b:%q ==> FALSE", i, erra.Error(), errb.Error())
			return false
		}
	}

	return true
}

// GoString implements fmt.GoStringer so the "%#v" format directive will
// generate consistent results during tests.
func (eb *ErrorBag) GoString() string {
	es := make([]string, len(eb.errs))
	for i, err := range eb.errs {
		es[i] = fmt.Sprintf("%#v", err)
	}

	return fmt.Sprintf("&errbag.ErrorBag{errs: []error{%s}}", strings.Join(es, ", "))
}
