// Package errors contains the error handling used by eventdb. It is largely
// inspired by (and contains code from) the upspin.io/errors package.
package errors

import (
	"bytes"
	"fmt"
	"log"
	"runtime"

	"github.com/findrandomevents/eventdb"
)

// Error is a domain error for eventdb. It contains fields used to populate
// parts of the error message. Some of the fields may be left unset.
type Error struct {
	// UserID is the name of the user attempting the operation.
	UserID eventdb.UserID
	// Op is the operation being performed, usually the name of the
	// method being invoked.
	Op Op
	// Kind is the class of error, such as permission failure, or "Other"
	// if its class is unknown or irrelevant.
	Kind Kind
	// The underlying error that triggered this one, if any.
	Err error

	// Stack information; used only when the 'debug' build tag is set.
	stack
}

func (e *Error) isZero() bool {
	return e.UserID == "" && e.Op == "" && e.Kind == 0 && e.Err == nil
}

// E builds an error value from its arguments.
// There must be at least one argument or E panics.
// The type of each argument determines its meaning.
// If more than one argument of a given type is presented,
// only the last one is recorded.
//
// If the error is printed, only those items that have been
// set to non-zero values will appear in the result.
//
// If Kind is not specified or Other, we set it to the Kind of
// the underlying error.
func E(args ...interface{}) error {
	if len(args) == 0 {
		panic("call to errors.E with no arguments")
	}
	e := &Error{}
	for _, arg := range args {
		switch arg := arg.(type) {
		case eventdb.UserID:
			e.UserID = arg
		case Op:
			e.Op = arg
		case string:
			e.Err = Str(arg)
		case Kind:
			e.Kind = arg
		case *Error:
			// Make a copy
			copy := *arg
			e.Err = &copy
		case error:
			e.Err = arg
		default:
			_, file, line, _ := runtime.Caller(1)
			log.Printf("errors.E: bad call from %s:%d: %v", file, line, args)
			return Errorf("unknown type %T, value %v in error call", arg, arg)
		}
	}

	// Populate stack information (only in debug mode).
	e.populateStack()

	prev, ok := e.Err.(*Error)
	if !ok {
		return e
	}

	// The previous error was also one of ours. Suppress duplications
	// so the message won't contain the same kind, file name or user name
	// twice.
	if prev.UserID == e.UserID {
		prev.UserID = ""
	}
	if prev.Kind == e.Kind {
		prev.Kind = Other
	}
	// If this error has Kind unset or Other, pull up the inner one.
	if e.Kind == Other {
		e.Kind = prev.Kind
		prev.Kind = Other
	}
	return e
}

// pad appends str to the buffer if the buffer already has some data.
func pad(b *bytes.Buffer, str string) {
	if b.Len() == 0 {
		return
	}
	b.WriteString(str)
}

func (e *Error) Error() string {
	b := new(bytes.Buffer)
	e.printStack(b)
	if e.Op != "" {
		pad(b, ": ")
		b.WriteString(string(e.Op))
	}
	if e.UserID != "" {
		pad(b, ", ")
		b.WriteString("user ")
		b.WriteString(string(e.UserID))
	}
	if e.Kind != 0 {
		pad(b, ": ")
		b.WriteString(e.Kind.String())
	}
	if e.Err != nil {
		if prevErr, ok := e.Err.(*Error); ok {
			if !prevErr.isZero() {
				pad(b, ":\n\t")
				b.WriteString(e.Err.Error())
			}
		} else {
			pad(b, ": ")
			b.WriteString(e.Err.Error())
		}
	}
	if b.Len() == 0 {
		return "no error"
	}
	return b.String()
}

// Op describes an operation. eg, "Service.EventGet"
type Op string

// Kind defines the kind of error this is, used for translating into HTTP status
// codes based on the type of error.
type Kind int

const (
	Other       Kind = iota // Unclassified error. This value is not printed in the error message.
	Invalid                 // Bad request
	NotLoggedIn             // Unauthorized.
	Permission              // Permission denied.
	NotExist                // Item does not exist.
	Exist                   // Item already exists.
	Internal                // Internal error or inconsistency.
)

func (k Kind) String() string {
	switch k {
	case Other:
		return "other error"
	case NotExist:
		return "item does not exist"
	case Exist:
		return "item already exists"
	case Permission:
		return "permission denied"
	case NotLoggedIn:
		return "not logged in"
	case Invalid:
		return "invalid request"
	case Internal:
		return "internal error"
	}
	return "unknown error kind"
}

// Recreate the errors.New functionality of the standard Go errors package
// so we can create simple text errors when needed.

// Str returns an error that formats as the given text. It is intended to
// be used as the error-typed argument to the E function.
func Str(text string) error {
	return &errorString{text}
}

// errorString is a trivial implementation of error.
type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}

// Errorf is equivalent to fmt.Errorf, but allows clients to import only this
// package for all error handling.
func Errorf(format string, args ...interface{}) error {
	return &errorString{fmt.Sprintf(format, args...)}
}

// Is reports whether err is an *Error of the given Kind.
// If err is nil then Is returns false.
func Is(kind Kind, err error) bool {
	e, ok := err.(*Error)
	if !ok {
		return false
	}
	if e.Kind != Other {
		return e.Kind == kind
	}
	if e.Err != nil {
		return Is(kind, e.Err)
	}
	return false
}

// Match compares its two error arguments. It can be used to check
// for expected errors in tests. Both arguments must have underlying
// type *Error or Match will return false. Otherwise it returns true
// iff every non-zero element of the first error is equal to the
// corresponding element of the second.
// If the Err field is a *Error, Match recurs on that field;
// otherwise it compares the strings returned by the Error methods.
// Elements that are in the second argument but not present in
// the first are ignored.
func Match(err1, err2 error) bool {
	e1, ok := err1.(*Error)
	if !ok {
		return false
	}
	e2, ok := err2.(*Error)
	if !ok {
		return false
	}
	if e1.UserID != "" && e2.UserID != e1.UserID {
		return false
	}
	if e1.Op != "" && e2.Op != e1.Op {
		return false
	}
	if e1.Kind != Other && e2.Kind != e1.Kind {
		return false
	}
	if e1.Err != nil {
		if _, ok := e1.Err.(*Error); ok {
			return Match(e1.Err, e2.Err)
		}
		if e2.Err == nil || e2.Err.Error() != e1.Err.Error() {
			return false
		}
	}
	return true
}
