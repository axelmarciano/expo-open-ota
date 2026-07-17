package store

import (
	"errors"
	"fmt"
	"strings"
)

var ErrNotSupportedInStatelessMode = errors.New("operation not supported in stateless mode")

// ErrWouldLeaveNoAdmin is returned by the guarded user delete/demote paths
// when the target is the last remaining admin: the write is refused
// atomically in SQL so the dashboard can never lock itself out, even under
// concurrent operations.
var ErrWouldLeaveNoAdmin = errors.New("operation refused: it would leave the dashboard without any admin account")

type ErrBranchHasActiveChannels struct {
	BranchName   string
	ChannelNames []string
}

func (e *ErrBranchHasActiveChannels) Error() string {
	channelsList := strings.Join(e.ChannelNames, ", ")
	return fmt.Sprintf("cannot delete branch %q because the following channels are still pointed to it: [%s]. Please unbind or delete these channels first.", e.BranchName, channelsList)
}

type ErrResourceAlreadyExists struct {
	Resource   string
	Identifier string
}

func (e *ErrResourceAlreadyExists) Error() string {
	return fmt.Sprintf("cannot create %s: a resource with identifier %q already exists.", e.Resource, e.Identifier)
}

type ErrResourceNotFound struct {
	Resource   string
	Identifier string
}

func (e *ErrResourceNotFound) Error() string {
	return fmt.Sprintf("%s with identifier %q not found.", e.Resource, e.Identifier)
}
