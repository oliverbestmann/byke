package spoke

import "os"

var debug = os.Getenv("SPOKE_DEBUG") == "true"
