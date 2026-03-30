package pages

import (
	"fmt"
	"strings"
)

// validateDescription checks that a plain-text description (HTML already
// stripped) meets minimum quality requirements.  It returns a user-facing
// error message on failure or nil when the text is acceptable.
func validateDescription(plainText string) error {
	plainText = strings.TrimSpace(plainText)
	if plainText == "" {
		return fmt.Errorf("A schematic must have a description")
	}

	words := strings.Fields(plainText)
	if len(words) < 5 {
		return fmt.Errorf("Please write at least 5 words describing your schematic")
	}

	// Check for any single word exceeding 30 characters (keyboard mashing).
	for _, w := range words {
		if len(w) > 30 {
			return fmt.Errorf("Your description contains invalid text. Please write a real description.")
		}
	}

	// Count word frequencies.
	freq := make(map[string]int, len(words))
	for _, w := range words {
		freq[strings.ToLower(w)]++
	}

	// All words identical.
	if len(freq) == 1 {
		return fmt.Errorf("Please write a real description of your schematic")
	}

	// More than half the words are repeated (i.e. appear more than once).
	repeated := 0
	for _, count := range freq {
		if count > 1 {
			repeated += count
		}
	}
	if repeated > len(words)/2 {
		return fmt.Errorf("Please write a more detailed description of your schematic")
	}

	return nil
}
