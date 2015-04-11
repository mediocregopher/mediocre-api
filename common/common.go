// Package common is used to hold things that multiple packags within the
// mediocre-api toolkit will need to use or reference
package common

// ExpectedErr is an implementation of the error interface which will be used to
// indicate that the error being returned is an expected one and can sent back
// to the client
type ExpectedErr struct {
	Code int
	Err  string
}

// Error implements the error interface
func (e ExpectedErr) Error() string {
	return e.Err
}
