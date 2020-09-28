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

package exechelper

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// CmdFunc function for setting exec.Cmd parameters.
type CmdFunc func(cmd *exec.Cmd) error

// Option - expresses optional behavior for exec.Cmd
type Option struct {
	// Context - context (if any) for running the exec.Cmd
	Context context.Context
	// SIGTERM grace period
	GracePeriod time.Duration
	// CmdFunc to be applied to the exec.Cmd
	CmdOption CmdFunc
}

// CmdOption - convenience function for producing an Option that only has an Option.CmdOption
func CmdOption(cmdFunc CmdFunc) *Option {
	return &Option{CmdOption: cmdFunc}
}

// WithContext - option for setting the context.Context for running the exec.Cmd
func WithContext(ctx context.Context) *Option {
	return &Option{Context: ctx}
}

// WithArgs - appends additional args to cmdStr
//            useful for ensuring correctness when you start from
//            args []string rather than from a cmdStr to be parsed
func WithArgs(args ...string) *Option {
	return CmdOption(func(cmd *exec.Cmd) error {
		cmd.Args = append(cmd.Args, args...)
		return nil
	})
}

// WithDir - Option that will create the requested dir if it does not exist and set exec.Cmd.Dir = dir
func WithDir(dir string) *Option {
	return CmdOption(func(cmd *exec.Cmd) error {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0750); err != nil {
				return err
			}
		}
		cmd.Dir = dir
		return nil
	})
}

// WithStdin - option to set exec.Cmd.Stdout
func WithStdin(reader io.Reader) *Option {
	return CmdOption(func(cmd *exec.Cmd) error {
		cmd.Stdin = reader
		return nil
	})
}

// WithStdout - option to provide a writer to receive exec.Cmd.Stdout
//              if multiple WithStdout options are received, they are combined
//              with an io.Multiwriter
func WithStdout(writer io.Writer) *Option {
	return CmdOption(func(cmd *exec.Cmd) error {
		if cmd.Stdout == nil {
			cmd.Stdout = writer
			return nil
		}
		cmd.Stdout = io.MultiWriter(cmd.Stdout, writer)
		return nil
	})
}

// WithStderr - option to provide a writer to receive exec.Cmd.Stderr
//              if multiple WithStderr options are received, they are combined
//              with an io.Multiwriter
func WithStderr(writer io.Writer) *Option {
	return CmdOption(func(cmd *exec.Cmd) error {
		if cmd.Stderr == nil {
			cmd.Stderr = writer
			return nil
		}
		cmd.Stderr = io.MultiWriter(cmd.Stderr, writer)
		return nil
	})
}

// WithEnvirons - add entries to exec.Cmd.Env as a series of "key=value" strings
// Example: WithEnvirons("key1=value1","key2=value2",...)
func WithEnvirons(environs ...string) *Option {
	var envs []string
	for _, env := range environs {
		kv := strings.SplitN(env, "=", 2)
		if len(kv) != 2 {
			return CmdOption(func(cmd *exec.Cmd) error {
				return errors.Errorf("environs variable %q is not properly formated as key=value", env)
			})
		}
		envs = append(envs, kv[0], kv[1])
	}
	return WithEnvKV(envs...)
}

// WithEnvKV - add entries to exec.Cmd as a series key,value pairs in a list of strings
// Existing instances of 'key' will be overwritten
// Example: WithEnvKV(key1,value2,key2,value2...)
func WithEnvKV(envs ...string) *Option {
	return CmdOption(func(cmd *exec.Cmd) error {
		if len(envs)%2 != 0 {
			return errors.Errorf("WithEnvKV requires an even number of arguments, odd number provided")
		}
		for i := 0; i < len(envs); i += 2 {
			prefix := envs[i] + "="
			value := prefix + envs[i+1]
			for j, env := range cmd.Env {
				if strings.HasPrefix(env, prefix) {
					cmd.Env[j] = value
					continue
				}
			}
			cmd.Env = append(cmd.Env, value)
		}
		return nil
	})
}

// WithEnvMap - add entries to exec.Cmd from envMap
// Existing instances of 'key' will be overwritten
// Example: WithEnvKV(map[string]string{key1:value1,key2:value2})
func WithEnvMap(envMap map[string]string) *Option {
	var envs []string
	for k, v := range envMap {
		envs = append(envs, k, v)
	}
	return WithEnvKV(envs...)
}

// WithGracePeriod - will send a SIGTERM when ctx.Done() and wait up to gracePeriod before
// SIGKILLing the process.
func WithGracePeriod(gracePeriod time.Duration) *Option {
	return &Option{GracePeriod: gracePeriod}
}
