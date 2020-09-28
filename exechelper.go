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
	"syscall"
	"time"

	"github.com/google/shlex"
	"github.com/pkg/errors"
)

// Run - Creates a exec.Cmd using cmdStr.  Runs exec.Cmd.Run and returns the resulting error
func Run(cmdStr string, options ...*Option) error {
	return <-Start(cmdStr, options...)
}

// Start - Creates an exec.Cmd cmdStr.  Runs exec.Cmd.Start.
func Start(cmdStr string, options ...*Option) <-chan error {
	errCh := make(chan error, 1)

	// Extract context from options
	optionCtx := extractContextFromOptions(options)

	// Extract graceperiod from options
	graceperiod, err := extractGracePeriodFromOptions(optionCtx, options)
	if err != nil {
		errCh <- err
		close(errCh)
		return errCh
	}

	// By default, the context passed to StartContext (ie, cmdCtx) is the same as the context we got from the options
	// (ie, optionsCtx)
	cmdCtx := optionCtx

	// But if we have a graceperiod, we need a separate cmdCtx and cmdCancel so we can insert our SIGTERM
	// between the optionsCtx.Done() and time.After(graceperiod) before *actually* canceling the cmdCtx
	// and thus sending SIGKILL to the cmd
	var cmdCancel context.CancelFunc
	if graceperiod != 0 {
		cmdCtx, cmdCancel = context.WithCancel(context.Background())
	}

	cmd, err := constructCommand(cmdCtx, cmdStr, options)
	if err != nil {
		errCh <- err
		close(errCh)
		if cmdCancel != nil {
			cmdCancel()
		}
		return errCh
	}

	// Start the *exec.Cmd
	if err = cmd.Start(); err != nil {
		errCh <- err
		close(errCh)
		if cmdCancel != nil {
			cmdCancel()
		}
		return errCh
	}

	// By default, the error channel we send any error from the wait to (waitErrCh) is the one we return (errCh)
	waitErrCh := errCh

	// But if we have a graceperiod and a cmdCancel, we need a distinct waitErrCh from the one we return,
	// so that we can select on waitErrCh after sending SIGTERM and then forward any errors to errCh
	if cmdCancel != nil && graceperiod > 0 {
		waitErrCh = make(chan error, len(errCh))
	}

	// Collect the wait
	go func(waitErrCh chan error) {
		if err := cmd.Wait(); err != nil {
			waitErrCh <- err
		}
		close(waitErrCh)
	}(waitErrCh)

	// Handle SIGTERM and graceperiod
	if cmdCancel != nil && graceperiod > 0 {
		go handleGracePeriod(optionCtx, cmd, cmdCancel, graceperiod, waitErrCh, errCh)
	}

	return errCh
}

func extractGracePeriodFromOptions(ctx context.Context, options []*Option) (time.Duration, error) {
	var graceperiod time.Duration
	for _, option := range options {
		if option.GracePeriod != 0 {
			graceperiod = option.GracePeriod
			if ctx == nil {
				return 0, errors.New("graceperiod cannot be set without WithContext option")
			}
		}
	}
	return graceperiod, nil
}

func extractContextFromOptions(options []*Option) context.Context {
	// Set the context
	var optionCtx context.Context
	for _, option := range options {
		if option.Context != nil {
			optionCtx = option.Context
		}
	}
	return optionCtx
}

func constructCommand(ctx context.Context, cmdStr string, options []*Option) (*exec.Cmd, error) {
	// Construct the command args
	args, err := shlex.Split(cmdStr)
	if err != nil {
		return nil, err
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
			if err := option.CmdOption(cmd); err != nil {
				return nil, err
			}
		}
	}
	return cmd, nil
}

func handleGracePeriod(optionCtx context.Context, cmd *exec.Cmd, cmdCancel context.CancelFunc, graceperiod time.Duration, waitErrCh <-chan error, errCh chan<- error) {
	// Wait for the optionCtx to be done
	<-optionCtx.Done()

	// Send SIGTERM
	_ = cmd.Process.Signal(syscall.SIGTERM)

	// Wait for either the waitErrCh to be closed or have an error (ie, cmd exited) or graceperiod
	// either way
	select {
	case <-waitErrCh:
	case <-time.After(graceperiod):
	}
	// Cancel the cmdCtx passed to exec.StartContext
	cmdCancel()

	// Move all errors from waitErrCh to errCh
	for err := range waitErrCh {
		errCh <- err
	}
	// Close errCh
	close(errCh)
}

// Output - Creates a exec.Cmd using cmdStr.  Runs exec.Cmd.Output and returns the resulting output as []byte and error
func Output(cmdStr string, options ...*Option) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	options = append(options, WithStdout(buffer))
	if err := Run(cmdStr, options...); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// CombinedOutput - Creates a exec.Cmd using cmdStr.  Runs exec.Cmd.CombinedOutput and returns the resulting output as []byte and error
func CombinedOutput(cmdStr string, options ...*Option) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	options = append(options, WithStdout(buffer), WithStderr(buffer))
	if err := Run(cmdStr, options...); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
