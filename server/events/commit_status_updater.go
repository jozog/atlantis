// Copyright 2017 HootSuite Media Inc.
//
// Licensed under the Apache License, Version 2.0 (the License);
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an AS IS BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Modified hereafter by contributors to runatlantis/atlantis.
//
package events

import (
	"fmt"
	"strings"

	"github.com/runatlantis/atlantis/server/events/models"
	"github.com/runatlantis/atlantis/server/events/vcs"
)

//go:generate pegomock generate -m --use-experimental-model-gen --package mocks -o mocks/mock_commit_status_updater.go CommitStatusUpdater

// CommitStatusUpdater updates the status of a commit with the VCS host. We set
// the status to signify whether the plan/apply succeeds.
type CommitStatusUpdater interface {
	// Update updates the status of the head commit of pull.
	Update(repo models.Repo, pull models.PullRequest, status vcs.CommitStatus, command CommandName, ctx *CommandContext, workspace string) error
	// UpdateProjectResult updates the status of the head commit given the
	// state of response.
	UpdateProjectResult(ctx *CommandContext, commandName CommandName, res CommandResult) error
}

// DefaultCommitStatusUpdater implements CommitStatusUpdater.
type DefaultCommitStatusUpdater struct {
	Client vcs.ClientProxy
}

// Update updates the commit status.
func (d *DefaultCommitStatusUpdater) Update(repo models.Repo, pull models.PullRequest, status vcs.CommitStatus, command CommandName, ctx *CommandContext, workspace string) error {
	description := fmt.Sprintf("%s %s", strings.Title(command.String()), strings.Title(status.String()))

	if command == Plan && status == vcs.Success {
		status = vcs.Pending
	}

	ctx.Log.Debug("Updating status %s/%s on %s", strings.Title(status.String()), description, workspace)

	return d.Client.UpdateStatus(repo, pull, status, description, workspace)
}

// UpdateProjectResult updates the commit status based on the status of res.
func (d *DefaultCommitStatusUpdater) UpdateProjectResult(ctx *CommandContext, commandName CommandName, res CommandResult) error {
	if res.Error != nil || res.Failure != "" {
		// TODO: iterate project/workspace and send failure on each one
	} else {
		for _, p := range res.ProjectResults {
			d.Update(ctx.BaseRepo, ctx.Pull, p.Status(), commandName, ctx, p.Workspace)
		}
	}
	return nil
}

func (d *DefaultCommitStatusUpdater) worstStatus(ss []vcs.CommitStatus) vcs.CommitStatus {
	for _, s := range ss {
		if s == vcs.Failed {
			return vcs.Failed
		}
	}
	return vcs.Success
}
