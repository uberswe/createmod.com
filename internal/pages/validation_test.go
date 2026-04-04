package pages

import (
	"testing"
)

func Test_validateDescription(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid description",
			input:   "A blue train with passenger carriages and steam engine",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
			errMsg:  "A schematic must have a description",
		},
		{
			name:    "whitespace only",
			input:   "   \t\n  ",
			wantErr: true,
			errMsg:  "A schematic must have a description",
		},
		{
			name:    "too few words",
			input:   "nice train here",
			wantErr: true,
			errMsg:  "Please write at least 5 words describing your schematic",
		},
		{
			name:    "exactly 5 words passes",
			input:   "A small red steam train",
			wantErr: false,
		},
		{
			name:    "all words identical",
			input:   "test test test test test",
			wantErr: true,
			errMsg:  "Please write a real description of your schematic",
		},
		{
			name:    "keyboard mashing long word",
			input:   "this is a asasasdadsdsdsdsdsdsdsdsdsdsdsdsd description for testing",
			wantErr: true,
			errMsg:  "Your description contains invalid text. Please write a real description.",
		},
		{
			name:    "low unique word ratio (spam)",
			input:   "go go go go go go go go go go go go go go go go go go go now",
			wantErr: true,
			errMsg:  "Please write a more detailed description of your schematic",
		},
		{
			name:    "borderline unique ratio passes",
			input:   "hello world foo bar baz qux",
			wantErr: false,
		},
		{
			name:    "real description with some repeated words",
			input:   "This is a train that runs on rails and carries passengers to the station",
			wantErr: false,
		},
		{
			name:    "long real description with naturally repeated words",
			input:   "A schematic that produces all the dyes Since I couldnt find a schematic that processed all the dyes in one schematic i decided to upload my own Items process all in one chunk",
			wantErr: false,
		},
		{
			name:    "long description skips gibberish checks",
			input:   "This is a very long and detailed description of my schematic that goes well over the two hundred character limit so the gibberish detector should not run and this should pass even with a superlongwordthatexceedsthirtychars inside it somewhere",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDescription(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %q", err.Error())
				}
			}
		})
	}
}
