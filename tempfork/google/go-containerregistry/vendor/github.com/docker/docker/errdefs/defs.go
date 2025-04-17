// Copyright 2025 AUTHORS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errdefs // import "github.com/docker/docker/errdefs"

// ErrNotFound signals that the requested object doesn't exist
type ErrNotFound interface {
	NotFound()
}

// ErrInvalidParameter signals that the user input is invalid
type ErrInvalidParameter interface {
	InvalidParameter()
}

// ErrConflict signals that some internal state conflicts with the requested action and can't be performed.
// A change in state should be able to clear this error.
type ErrConflict interface {
	Conflict()
}

// ErrUnauthorized is used to signify that the user is not authorized to perform a specific action
type ErrUnauthorized interface {
	Unauthorized()
}

// ErrUnavailable signals that the requested action/subsystem is not available.
type ErrUnavailable interface {
	Unavailable()
}

// ErrForbidden signals that the requested action cannot be performed under any circumstances.
// When a ErrForbidden is returned, the caller should never retry the action.
type ErrForbidden interface {
	Forbidden()
}

// ErrSystem signals that some internal error occurred.
// An example of this would be a failed mount request.
type ErrSystem interface {
	System()
}

// ErrNotModified signals that an action can't be performed because it's already in the desired state
type ErrNotModified interface {
	NotModified()
}

// ErrNotImplemented signals that the requested action/feature is not implemented on the system as configured.
type ErrNotImplemented interface {
	NotImplemented()
}

// ErrUnknown signals that the kind of error that occurred is not known.
type ErrUnknown interface {
	Unknown()
}

// ErrCancelled signals that the action was cancelled.
type ErrCancelled interface {
	Cancelled()
}

// ErrDeadline signals that the deadline was reached before the action completed.
type ErrDeadline interface {
	DeadlineExceeded()
}

// ErrDataLoss indicates that data was lost or there is data corruption.
type ErrDataLoss interface {
	DataLoss()
}
