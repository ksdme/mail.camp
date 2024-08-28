package utils

import (
	"fmt"
	"io"
	"strings"
)

func AskConsent(s io.ReadWriter, prompt string) bool {
	fmt.Fprint(s, prompt)

	var consent string
	fmt.Fscanf(s, "%s", &consent)
	consent = strings.ToLower(consent)
	consent = strings.TrimSpace(consent)

	return consent == "yes"
}
