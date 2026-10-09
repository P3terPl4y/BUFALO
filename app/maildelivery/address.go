package maildelivery

import (
	"net/mail"
	"strings"
)

func envelopeAddress(value string) (string, error) {
	address, err := mail.ParseAddress(value)
	if err != nil {
		return "", err
	}
	return address.Address, nil
}
func containsPlain(value string) bool {
	for _, item := range strings.Fields(value) {
		if strings.EqualFold(item, "PLAIN") {
			return true
		}
	}
	return false
}
