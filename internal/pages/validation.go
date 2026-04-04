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

	// Skip gibberish checks for longer descriptions — a user who wrote
	// 200+ characters is clearly making a genuine effort.
	if len(plainText) <= 200 {
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

		// If fewer than 15% of words are unique the text is likely
		// copy-pasted spam or meaningless repetition.  Normal English prose
		// typically has 50%+ unique words even in longer passages.
		if float64(len(freq)) < float64(len(words))*0.15 {
			return fmt.Errorf("Please write a more detailed description of your schematic")
		}
	}

	return nil
}
