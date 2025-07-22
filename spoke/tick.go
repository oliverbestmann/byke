package spoke

// Tick counts system executions in a world
// TODO At 1000 systems per frame and 60 fps this will overflow after ~20h
type Tick uint32

const NoTick Tick = 0xffffffff
