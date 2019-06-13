// Copyright © 2018 Timothy E. Peoples <eng@toolman.org>
//
// This program is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public License
// as published by the Free Software Foundation; either version 2
// of the License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

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

//----------------------------------------------------------------------------

type newTestcase struct {
	err    error
	others []interface{}
}

func (tc *newTestcase) want() *ErrorBag {
	if tc.err == nil {
		return nil
	}

	var eb *ErrorBag

	if e, ok := tc.err.(errBagger); ok {
		eb = e.errBag()
	} else {
		eb = new(ErrorBag)
		eb.Add(tc.err)
	}

	if len(tc.others) == 0 {
		return eb
	}

	for _, o := range tc.others {
		if err, ok := o.(error); ok && err != nil {
			eb.Add(err)
		} else if ef, ok := o.(func() error); ok {
			if err := ef(); err != nil {
				eb.Add(err)
			}
		} else if ef, ok := o.(ErrorFunc); ok {
			if err := ef(); err != nil {
				eb.Add(err)
			}
		}
	}

	return eb
}

func (tc *newTestcase) test(t *testing.T) {
	want := tc.want()
	if got := New(tc.err, tc.others...); !reflect.DeepEqual(got, want) {
		t.Errorf("New(%q, []interface{}{%#v}...) := (%#v); Wanted(%#v)", tc.err, tc.others, got, want)
	}
}

func mkNewTestcase(err error, others ...interface{}) *newTestcase {
	return &newTestcase{err, others}
}

func newTestcaseLabel(err interface{}, others ...interface{}) string {
	parts := make([]string, len(others)+2)
	if err == nil {
		parts[0] = "nilErr"
	} else if _, ok := err.(error); ok {
		parts[0] = "anErr"
	} else {
		panic(fmt.Sprintf("bad value: %v", err))
	}

	funclabel := func(f func() error) string {
		if f == nil {
			return "noFunc"
		}
		if err := f(); err != nil {
			return "errFunc"
		}
		return "nilFunc"
	}

	if others == nil {
		parts[1] = "none"
	} else if len(others) == 0 {
		parts[1] = "empty"
	} else {
		for i, o := range others {
			var part string
			if o == nil {
				part = "nilErr"
			} else {
				switch v := o.(type) {
				case error:
					part = "anErr"

				case ErrorFunc:
					part = funclabel(v)
				}
			}

			parts[i+1] = part
		}
	}

	return strings.Join(parts, "-")
}

func TestNew(t *testing.T) {
	var (
		nilErr  error
		anErr   = errors.New("an error")
		nilFunc = func() error { return nil }
		errFunc = func() error { return anErr }
	)

	// TODO: This does not test the case that no ErrorFunc should be called if
	//       the primary err is nil.
	ovals := []interface{}{nil, nilErr, anErr, nilFunc, errFunc}
	cases := make(map[string]*newTestcase)

	var fill func([]interface{}, int, func([]interface{}))

	fill = func(list []interface{}, pos int, emit func([]interface{})) {
		for _, ov := range ovals {
			list[pos] = ov
			if pos == len(list)-1 {
				emit(list)
			} else {
				fill(list, pos+1, emit)
			}
		}
	}

	for _, err := range []error{nilErr, anErr} {
		cases[newTestcaseLabel(err)] = mkNewTestcase(err)

		// builds all permutations of "ovals" at length 1, 2 and 3
		for n := 1; n <= 3; n++ {
			fill(make([]interface{}, n), 0, func(a []interface{}) {
				cases[newTestcaseLabel(err, a...)] = mkNewTestcase(err, a...)
			})
		}
	}

	for name, tc := range cases {
		t.Run(name, tc.test)
	}
}
