/**
 * Copyright 2025 AUTHORS
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

export const State = {
  Running: "running",
  Starting: "starting",
  Stopping: "stopping",
  Stopped: "stopped",
  Partial: "partial",
  Unknown: "unknown",
};

export const statusToState = (status) => {
  const components = Object.values(status.components);
  const states = components.map((c) => c.status);
  const uniqueStates = new Set(states);

  if (uniqueStates.has(State.Stopping)) {
    return State.Stopping;
  }
  if (uniqueStates.has(State.Starting)) {
    return State.Starting;
  }

  if (uniqueStates.size === 1) {
    switch (states[0]) {
      case State.Running:
        return State.Running;
      case State.Stopped:
        return State.Stopped;
      default:
        return State.Unknown;
    }
  }

  return State.Partial;
};

export const stateToColor = (state) => {
  switch (state) {
    case State.Running:
      return "green";
    case State.Starting:
      return "indigo";
    case State.Stopping:
      return "orange";
    case State.Stopped:
      return "red";
    case State.Partial:
      return "yellow";
    default:
      return "gray";
  }
};
