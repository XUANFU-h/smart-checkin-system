package controller

import (
	"os"
	"strings"
)

func isEmergencySecureKeyValid(secureKey string) bool {
	expected := strings.TrimSpace(os.Getenv("SUPER_ADMIN_SECURE_KEY"))
	return expected != "" && secureKey != "" && secureKey == expected
}
