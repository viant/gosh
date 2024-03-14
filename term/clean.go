package term

import "regexp"

var specialSequence = regexp.MustCompile(`\x1b[\[\]()#;?]*(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]`)

// Clean removes a broader range of terminal control characters from a string.
func Clean(input string) string {
	// This pattern is more comprehensive, matching escape sequences that start with "\x1b",
	// followed by any character (mostly "[", but could be others for different types of sequences),
	// followed by any sequence of numbers, letters, and semicolons, and ending with a letter from A to Z (upper or lower case).
	// It aims to catch a wide variety of control sequences, including those for cursor movement, screen clearing, style changes, etc.
	// Replace occurrences of the pattern with an empty string, effectively removing them.
	cleanedString := specialSequence.ReplaceAllString(input, "")
	return cleanedString
}
