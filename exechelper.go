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

// Package exechelper provides a wrapper around cmd.Exec that makes it easier to use
package exechelper

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/google/shlex"
)

// Run - Creates a exec.Prepare using cmdStr.  Runs exec.Prepare.Run and returns the resulting error
func Run(cmdStr string, options ...*Option) error {
	return <-Start(cmdStr, options...)
}

// Start - Creates an exec.Prepare cmdStr.  Runs exec.Prepare.Start.
func Start(cmdStr string, options ...*Option) <-chan error {
	errCh := make(chan error, 1)

	// Set the context
	var ctx context.Context
	for _, option := range options {
		ctx = option.Context
	}

	// Construct the command args
	args, err := shlex.Split(cmdStr)
	if err != nil {
		errCh <- err
		close(errCh)
		return errCh
	}

	// Create the *exec.Cmd
	var cmd *exec.Cmd
	switch ctx {
	case nil:
		cmd = exec.Command(args[0], args[1:]...) // #nosec
	default:
		cmd = exec.CommandContext(ctx, args[0], args[1:]...) // #nosec
	}

	// Apply the options to the *exec.Cmd
	for _, option := range options {
		// Apply the CmdOptions
		if option.CmdOption != nil {
			if err = option.CmdOption(cmd); err != nil {
				errCh <- err
				close(errCh)
				return errCh
			}
		}
	}

	// Start the *exec.Cmd
	if err = cmd.Start(); err != nil {
		errCh <- err
		close(errCh)
		return errCh
	}

	// Collect the wait
	go func(chan error) {
		if err := cmd.Wait(); err != nil {
			errCh <- err
		}
		close(errCh)
	}(errCh)

	return errCh
}

// Output - Creates a exec.Prepare using cmdStr.  Runs exec.Prepare.Output and returns the resulting output as []byte and error
func Output(cmdStr string, options ...*Option) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	options = append(options, WithStdout(buffer))
	if err := Run(cmdStr, options...); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// CombinedOutput - Creates a exec.Prepare using cmdStr.  Runs exec.Prepare.CombinedOutput and returns the resulting output as []byte and error
func CombinedOutput(cmdStr string, options ...*Option) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	options = append(options, WithStdout(buffer), WithStderr(buffer))
	if err := Run(cmdStr, options...); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
