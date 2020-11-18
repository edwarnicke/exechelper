// Copyright (c) 2020 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build linux

package exechelper

import (
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
	"github.com/vishvananda/netns"
)

// WithOnDeathSignalChildren - set the signal that will be sent to children of process on processes death
// (only available on linux)
func WithOnDeathSignalChildren(signal syscall.Signal) *Option {
	return CmdOption(func(cmd *exec.Cmd) error {
		cmd.SysProcAttr.Pdeathsig = signal
		return nil
	})
}

// WithNetNS - run the cmd in the network namespace (netNS) specified by handle.
func WithNetNS(handle netns.NsHandle) *Option {
	originalNetNs, err := netns.Get()
	return &Option{
		CmdOption: func(cmd *exec.Cmd) error {
			if err != nil {
				return errors.Wrap(err, "unable to retrieve original netns.Handle")
			}
			return errors.Wrap(netns.Set(handle), "unable to set to requested netns.Handle")
		},
		PostRunOption: func(cmd *exec.Cmd) error {
			if err != nil {
				_ = netns.Set(originalNetNs)
				return err
			}
			return errors.Wrap(netns.Set(originalNetNs), "unable to set to restore original netns.Handle")
		},
	}
}
