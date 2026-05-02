package byke

import (
	"os"
	"strings"

	"github.com/oliverbestmann/puffin-go"
)

func init() {
	if addr := os.Getenv("PUFFIN_ADDR"); addr != "" {
		puffin.Enable(addr)
		return
	}

	enabled := os.Getenv("PUFFIN_ENABLE")
	if enabled == "1" || strings.ToLower(enabled) == "true" {
		puffin.Enable("127.0.0.1:8585")
	}
}
