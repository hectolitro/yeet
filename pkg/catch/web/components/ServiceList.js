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

import React, { useContext } from "react";
import clsx from "clsx";
import htm from "htm";

import { ServiceContext, useService } from "../context/ServiceContext.js";
import { State, stateToColor, statusToState } from "../lib/status.js";

const html = htm.bind(React.createElement);

const StateIndicator = ({ serviceName, state }) => {
  const color = stateToColor(state);
  return html`
    <span
      className=${clsx("w-3 h-3 rounded-full", `bg-${color}-500`)}
      aria-label="${serviceName} is ${state}"
      alt="${serviceName} status indicator: ${state}"
    ></span>
  `;
};

const styles = {
  container:
    "flex cursor-pointer select-none flex-col rounded px-3 py-2 active:translate-y-0.5",
  default: "border border-slate-500/80 bg-slate-900/20",
  hover: "hover:border-green-600/60 hover:bg-green-900/20 hover:text-green-500",
  selected: "!border-green-600/80 !bg-green-900/30 text-green-500",
};

const Service = ({ serviceId, selected }) => {
  const { serviceName, status, setSelectedService } = useService(serviceId);
  let state = State.Unknown;
  if (status) {
    state = statusToState(status);
  }
  return html`<div
    className=${clsx(
      styles.container,
      styles.default,
      !selected && styles.hover,
      selected && styles.selected
    )}
    onClick=${() => setSelectedService()}
  >
    <div className="flex items-center justify-between">
      <span className="text-md truncate">${serviceName}</span>
      <${StateIndicator} serviceName=${serviceName} state=${state} />
    </div>
  </div>`;
};

const ServiceList = () => {
  const { state } = useContext(ServiceContext);
  const { selectedService } = state;

  const sortedServices = Object.entries(state.services).sort(([a], [b]) => {
    if (a === "catch") return -1;
    if (b === "catch") return 1;
    return a.localeCompare(b);
  });

  return html`<ul role="list" className="flex flex-col gap-y-3">
    ${sortedServices.map(
      ([serviceId]) =>
        html`<li>
          <${Service}
            key=${serviceId}
            serviceId=${serviceId}
            selected=${serviceId === selectedService}
          />
        </li>`
    )}
  </ul>`;
};
export default ServiceList;
