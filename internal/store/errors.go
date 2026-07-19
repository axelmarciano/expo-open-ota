package store

import (
	"errors"
	"fmt"
	"strings"
)

var ErrNotSupportedInStatelessMode = errors.New("operation not supported in stateless mode")

// ErrPublishBlockedByActiveRollout and ErrRolloutSupersededByNewerUpdate are the
// two refusal reasons of the conditional MarkUpdateAsChecked stamp: the guard runs
// inside the UPDATE itself so the publish/activation races close transactionally,
// and the store disambiguates the 0-rows result into one of these.
var ErrPublishBlockedByActiveRollout = errors.New("update cannot become visible: a progressive rollout is active on its branch and runtime version")
var ErrRolloutSupersededByNewerUpdate = errors.New("rollout activation refused: a newer update was published during the upload")

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

type ErrBranchInActiveRollout struct {
	BranchName   string
	ChannelNames []string
}

func (e *ErrBranchInActiveRollout) Error() string {
	channelsList := strings.Join(e.ChannelNames, ", ")
	return fmt.Sprintf("cannot delete branch %q because it is serving an active rollout on the following channels: [%s]. Promote or revert these rollouts first.", e.BranchName, channelsList)
}

type ErrChannelHasActiveRollout struct {
	ChannelName string
}

func (e *ErrChannelHasActiveRollout) Error() string {
	return fmt.Sprintf("cannot change the branch mapping of channel %q while it has an active rollout. Promote or revert the rollout first.", e.ChannelName)
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
