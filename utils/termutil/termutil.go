package termutil

import "os"

var noColor = os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"

// NoColor returns true if color output should be disabled
func NoColor() bool {
	return noColor
}
