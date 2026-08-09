package byke

import "slices"

// CommandQueue handles queueing and execution of Command instances.
//
// Notes on when commands are applied:
//
// Running a schedule: Commands are applied after each system in the schedule
//
// TODO might change this to only apply when we have dependencies
// between a system that schedules commands and another one that
// has a dependency (even indirect) on the first one.
//
// Running a system using World.RunSystem: Commands created by the
// system are applied directly after the system is run.
//
// When an Event is triggered and runs Observers: All observer systems are first
// executed and write their commands to the command queue. Once all systems are
// executed, the pending commands (created by those observers) are applied.
//
// When a system execution was scheduled via Command: The system is executed
// during command execution using World.RunSystem. Same rules as defined above
// apply.
//
// All of the above imply that no commands are applied that were already
// on the command queue before the operation (run system, run schedule,
// event trigger, etc.) starts.
type CommandQueue struct {
	pending []Command
}

func (c *CommandQueue) Append(command Command) {
	c.pending = append(c.pending, command)
}

func (c *CommandQueue) AppendAll(commands []Command) {
	c.pending = append(c.pending, commands...)
}

func (c *CommandQueue) Checkpoint() Checkpoint {
	return Checkpoint(len(c.pending))
}

func (c *CommandQueue) DrainAt(checkpoint Checkpoint) []Command {
	if c.Checkpoint() == checkpoint {
		// no need to allocate if we have nothing queued
		return nil
	}

	buf := slices.Clone(c.pending[checkpoint:])

	// remove the commands we've just removed
	clear(c.pending[checkpoint:])
	c.pending = c.pending[:checkpoint]

	return buf
}

type Checkpoint int
