package localsvr

import (
	"regexp"
)

// safeIdentifierRegexp is a conservative matcher: letters, digits, underscore
var safeIdentifierRegexp = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")

func IsSafeIdentifier(s string) (isSafe bool) {
	if s == "" {
		goto end
	}
	isSafe = safeIdentifierRegexp.MatchString(s)
end:
	return isSafe
}
