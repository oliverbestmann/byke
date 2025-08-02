package spoke

import (
	"log/slog"
	"strconv"
)

type EntityId uint32

func (e EntityId) String() string {
	return strconv.Itoa(int(e))
}

func (e EntityId) LogValue() slog.Value {
	return slog.StringValue(e.String())
}
