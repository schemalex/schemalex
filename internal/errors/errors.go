package errors

import (
	daverr "github.com/pkg/errors"
)

type ignorableErr struct {
	err error
}

type ignorabler interface {
	Ignorable() bool
}

type causer interface {
	Cause() error
}

func (e ignorableErr) Error() string {
	if e.err != nil {
		return e.err.Error() + " (ignorable)"
	}
	return "(ignorable)"
}

func (e ignorableErr) Ignorable() bool {
	return true
}

func Ignorable(err error) error {
	return ignorableErr{err: err}
}

func IsIgnorable(err error) bool {
	for err != nil {
		if ignore, ok := err.(ignorabler); ok {
			return ignore.Ignorable()
		}

		if cerr, ok := err.(causer); ok {
			err = cerr.Cause()
		} else {
			return false
		}
	}
	return false
}

func New(s string) error {
	return daverr.New(s)
}

func Errorf(s string, args ...interface{}) error {
	return daverr.Errorf(s, args...)
}

func Wrap(err error, s string) error {
	return daverr.Wrap(err, s)
}

func Wrapf(err error, s string, args ...interface{}) error {
	return daverr.Wrapf(err, s, args...)
}
