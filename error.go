package thevent

import (
	"strconv"
	"strings"
)

// TypeError is used to signal an event or handler type mismatch/misconfiguration
type TypeError struct{ error }

// MultiTypeError combines/wraps multiple TypeErrors into a single error
type MultiTypeError []TypeError

func (mte MultiTypeError) Error() string {
	quoted := make([]string, 0, len(mte))
	for _, e := range mte {
		quoted = append(quoted, strconv.Quote(e.Error()))
	}
	return "MultiTypeError: [" + strings.Join(quoted, ", ") + "]"
}
