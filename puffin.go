package byke

import (
	"os"
	"strings"

	"github.com/oliverbestmann/puffin-go"
)

func init() {
	waitForClient := isTrue(os.Getenv("PUFFIN_WAIT"))

	if addr := os.Getenv("PUFFIN_ADDR"); addr != "" {
		puffin.Enable(addr, waitForClient)
		return
	}

	enabled := os.Getenv("PUFFIN_ENABLED")
	if isTrue(enabled) {
		puffin.Enable("127.0.0.1:8585", waitForClient)
	}
}

func isTrue(value string) bool {
	return value == "1" || strings.ToLower(value) == "true"
}
