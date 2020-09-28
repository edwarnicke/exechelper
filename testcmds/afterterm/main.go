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

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Catch the signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, []os.Signal{
		os.Interrupt,
		// More Linux signals here
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}...)
	log.Printf("starting...\n")
	var d time.Duration
	if len(os.Args) > 1 {
		var err error
		d, err = time.ParseDuration(os.Args[1])
		if err != nil {
			log.Fatalf("os.Args[1]: %q is not a valid duration", os.Args[1])
		}
	}

	sig := <-c
	log.Printf("received signal %q\n", sig)
	<-time.After(d)
	log.Printf("exiting after %q\n", d)
}
