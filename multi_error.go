package gwr

import (
	"fmt"
	"strings"
)

// MultiErr bundles more than one error together into a single error.
type MultiErr []error

// Error returns a string like "[E1, E2, ...]" where each Ex is the Error() of
// each error in the slice.
func (mienErrs MultiErr) Error() string {
	parts := make([]string, len(mienErrs))
	for i, err := range mienErrs {
		parts[i] = err.Error()
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

// AsError returns either: nil, the only error, or the MultiErr instance itself
// if there are 0, 1, or more errors in the slice respectively.  This method is
// useful for contexts that want to do a simple return
// MultiError(errs).AsError().
func (mienErrs MultiErr) AsError() error {
	switch len(mienErrs) {
	case 0:
		return nil
	case 1:
		return mienErrs[1]
	default:
		return mienErrs
	}
}
