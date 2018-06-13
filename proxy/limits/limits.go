// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !windows

// Package limits provides routines to check and enforce certain resource
// limits on the Cloud SQL client proxy process.
package limits

import (
	"fmt"
	"syscall"

	"github.com/GoogleCloudPlatform/cloudsql-proxy/logging"
)

var (
	// For overriding in unittests.
	syscallGetrlimit = syscall.Getrlimit
	syscallSetrlimit = syscall.Setrlimit
)

// SetupFDLimits ensures that the process running the Cloud SQL proxy can have
// at least wantFDs number of open file descriptors. It returns an error if it
// cannot ensure the same.
func SetupFDLimits(wantFDs uint64) error {
	rlim := &syscall.Rlimit{}
	if err := syscallGetrlimit(syscall.RLIMIT_NOFILE, rlim); err != nil {
		return fmt.Errorf("failed to read rlimit for max file descriptors: %v", err)
	}

	if rlim.Cur >= wantFDs {
		logging.Infof("current FDs rlimit set to %d, wanted limit is %d. Nothing to do here.", rlim.Cur, wantFDs)
		return nil
	}

	// Linux man page:
	// The soft limit is the value that the kernel enforces for the corre‐
	// sponding resource. The hard limit acts as a ceiling for the soft limit:
	// an unprivileged process may set only its soft limit to a value in the
	// range from 0 up to the hard limit, and (irreversibly) lower its hard
	// limit. A privileged process (under Linux: one with the CAP_SYS_RESOURCE
	// capability in the initial user namespace) may make arbitrary changes to
	// either limit value.
	if rlim.Max < wantFDs {
		// When the hard limit is less than what is requested, let's just give it a
		// shot, and if we fail, we fallback and try just setting the softlimit.
		rlim2 := &syscall.Rlimit{}
		rlim2.Max = wantFDs
		rlim2.Cur = wantFDs
		if err := syscallSetrlimit(syscall.RLIMIT_NOFILE, rlim2); err == nil {
			logging.Infof("Rlimits for file descriptors set to {%v}", rlim2)
			return nil
		}
	}

	rlim.Cur = wantFDs
	if err := syscallSetrlimit(syscall.RLIMIT_NOFILE, rlim); err != nil {
		return fmt.Errorf("failed to set rlimit {%v} for max file descriptors: %v", rlim, err)
	}

	logging.Infof("Rlimits for file descriptors set to {%v}", rlim)
	return nil
}
