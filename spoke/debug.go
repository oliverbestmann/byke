package spoke

import "os"

// TODO flip this once we are pretty happy with most of the things.
var debug = os.Getenv("DEBUG") != "false"
