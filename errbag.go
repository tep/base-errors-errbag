package errbag

import (
	"fmt"
	"sort"
)

// ErrorBag is a collector for multiple errors implemented in a fluent style
// (i.e. many of the methods return the ErrorBag instance upon which they
// were called). ErrorBag also implements Error so it is itself an error.
//
// ErrorBag takes extra care to not add itself as one of its errors but it
// cannot catch all cases; e.g. a call to its Errorf method with the current
// ErrorBag instance as one of the interface parameters is not easily detected.
type ErrorBag struct {
	errs    []error
	wrapper ErrorWrapper
}

// Error implements error for the ErrorBag eb. If eb contains only 1 error, the
// the results of its Error method are returned. If eb contains more then
// 1 error, then a message is returned indicating how many errors it encounted;
// the caller should use Errors or Visit to access the contained errors.
// When eb contains no errors an empty string is returned.
func (eb *ErrorBag) Error() string {
	if l := len(eb.errs); l == 1 {
		return eb.errs[0].Error()
	} else if l > 1 {
		return fmt.Sprintf("encountered %d errors", l)
	}
	return ""
}

// Errors returns the slice of errors currently contained in the ErrorBag eb.
func (eb *ErrorBag) Errors() []error {
	return []error(eb.errs)
}

// Sorted returns the errors contained in the ErrorBag eb sorted lexically.
func (eb *ErrorBag) Sorted() []error {
	errs := eb.Errors()
	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return errs
}

// ErrorOrNil returns eb if it contains any errors whatsoever, otherwise it
// returns nil.
func (eb *ErrorBag) ErrorOrNil() error {
	if len(eb.errs) > 0 {
		return eb
	}
	return nil
}

// AsError returns eb as an error
func (eb *ErrorBag) AsError() error {
	return eb
}

// HasErrors returns true if eb contains any errors. Otherwise it returns
// false.
func (eb *ErrorBag) HasErrors() bool {
	return len(eb.errs) > 0
}

// Size returns the number of errors currenting in the ErrorBag eb.
func (eb *ErrorBag) Size() int {
	return len(eb.errs)
}

// Visitor is the function reference passed to the Visit function or method.
type Visitor func(error)

// Visit first determines whether err is an instance of ErrorBag, or is a type
// which embeds an instance of ErrorBag. If so, then the Visitor function v is
// executed for each error contained therein. If err cannot be reduced to an
// ErrorBag, then no action is taken.
func Visit(err error, v Visitor) {
	if eb := AsErrorBag(err); eb != nil {
		eb.Visit(v)
	}
}

// Visit executes the given Visitor for each error currently in the ErrorBag eb.
func (eb *ErrorBag) Visit(v Visitor) {
	for _, err := range eb.errs {
		v(err)
	}
}

type errBagger interface {
	errBag() *ErrorBag
}

func (eb *ErrorBag) errBag() *ErrorBag {
	return eb
}

// AsErrorBag attempts to reduce err to its base ErrorBag and, if so, return
// it.  If err cannot be reduced to an ErrorBag, AsErrorBag returns nil.
func AsErrorBag(err error) *ErrorBag {
	if err != nil {
		if e, ok := err.(errBagger); ok {
			return e.errBag()
		}
	}

	return nil
}

// Add is used to add err to the ErrorBag eb. If err is nil, nothing is added.
// If err is equal to eb (or, is a type which has embedded eb) then err will
// not be added. However, if err is a separate and distinct instance of
// ErrorBag, then each of its errors will be added in turn to eb. In all cases,
// eb is returned, modified or not.
func (eb *ErrorBag) Add(err error) error {
	if err == nil {
		return eb
	}

	if oeb := AsErrorBag(err); oeb != nil {
		if oeb == eb {
			return eb
		}

		// if err is an ErrorBag (but it's not *this* ErrorBag),
		// merge all the errors from oeb into eb.
		return eb.Update(oeb.Errors())
	}

	eb.errs = append(eb.errs, err)
	return eb
}

// Update calls Add for each error in errs then returns eb.
func (eb *ErrorBag) Update(errs []error) error {
	for _, err := range errs {
		eb.Add(err)
	}
	return eb
}

// Errorf is a convenience function that is the same as:
// 		eb.Add(fmt.Errorf(msgs, a...))
func (eb *ErrorBag) Errorf(msg string, a ...interface{}) error {
	return eb.Add(fmt.Errorf(msg, a...))
}

// ErrorWrapper provides WrapError for wrapping errors
type ErrorWrapper interface {
	// WrapError returns err optionally wrapped with a different error type.
	WrapError(err error) error
}

// WithWrapper returns a new *ErrorBag that will use wrapper for calls to
// its Wrap method.
func WithWrapper(wrapper ErrorWrapper) *ErrorBag {
	return &ErrorBag{wrapper: wrapper}
}

// The ErrorWrapper method installs wrapper into an existing ErrorBag.
// Subsequent calls to its Wrap method will use the newly installed
// ErroWrapper. To disable future wrapping, pass a nil value to this
// method.
func (eb *ErrorBag) ErrorWrapper(wrapper ErrorWrapper) {
	eb.wrapper = wrapper
}

// Wrap passes err to the contained ErrorWrapper's WrapError func and then
// adds its return value to eb in the same manner as Add. Wrap then returns
// eb (modified or not).
//
// If err is nil, nothing is added and eb is returned.
//
// As with Add, if err equals eb (or is a type that embeds eb) then err will
// not be added to eb. However, if err is an instance of ErrorBag but is not
// eb, then each of its contained errors will, in turn, be wrapped using
// WrapError and then added to eb.
//
// Note that, by default, ErrorBag has no implementation of ErrorWrapper so
// its default behavior is equivalient to Add; to leverage Wrap, you must
// create an ErrorBag using the WithWrapper constructor or, if ErrorBag is
// embedded in aother type, you may call the ErrorWrapper method to assign
// a new ErrorWrapper interface to your type.
func (eb *ErrorBag) Wrap(err error) error {
	if err == nil {
		return eb
	}

	if eb.wrapper == nil {
		return eb.Add(err)
	}

	if oeb := AsErrorBag(err); oeb != nil {
		if oeb != eb {
			oeb.Visit(func(err error) {
				eb.Wrap(err)
			})
		}

		return eb
	}

	return eb.Add(eb.wrapper.WrapError(err))
}
