// This example demonstrates how to embed an ErrorBag into a type that
// implements a custom ErrorWrapper that will be called by the Wrap method.
package errbag_test

import (
	"errors"
	"fmt"

	"toolman.org/base/errors/errbag"
)

// A custom error type that will wrap errors passed to the Wrap method.
type customError struct{ error }

// GoString implements fmt.GoStringer for the %#v format directive.
// It is employed here to ensure consistent example output.
func (c *customError) GoString() string {
	return "&" + fmt.Sprintf("%T{error:%#v}", c, c.error)[1:]
}

// A custom ErrorWrapper
type wrapper struct{}

// WrapError implements errbag.ErrorWrapper
func (w *wrapper) WrapError(err error) error {
	if _, ok := err.(*customError); ok {
		return err
	}

	return &customError{err}
}

func Example_errbagWrap() {
	eb := errbag.WithWrapper(&wrapper{})
	err := errors.New("plain error")
	ce := &customError{errors.New("custom error")}
	eb.Wrap(err)
	eb.Wrap(ce)
	eb.Visit(func(err error) { fmt.Printf("%#v\n", err) })
	// Output:
	// &errbag_test.customError{error:&errors.errorString{s:"plain error"}}
	// &errbag_test.customError{error:&errors.errorString{s:"custom error"}}
}
