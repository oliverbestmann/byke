package arch

import (
	"strconv"
)

type EntityId uint32

func (e EntityId) String() string {
	return strconv.Itoa(int(e))
}
