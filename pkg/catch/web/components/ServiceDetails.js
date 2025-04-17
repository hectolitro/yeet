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

import React from "react";
import htm from "htm";

import { State, statusToState } from "../lib/status.js";
import {
  StartIcon,
  StopIcon,
  RestartIcon,
  GearIcon,
  EditIcon,
} from "./Icons.js";
import { HoverTactileButton } from "./Buttons.js";
import { useSelectedService } from "../context/ServiceContext.js";
import TerminalTabs from "./TerminalTabs.js";

const html = htm.bind(React.createElement);

const ServiceControls = ({ disabled, state, onCommandChange }) => {
  const isTransitioning = ![
    State.Partial,
    State.Stopped,
    State.Running,
  ].includes(state);
  const disableStartButton =
    disabled || isTransitioning || state === State.Running;
  const disableStopButton =
    disabled || isTransitioning || state === State.Stopped;
  const disableRestartButton = disabled || disableStopButton;

  return html`
    <${HoverTactileButton} onClick=${() => onCommandChange("start")}
      disabled=${disableStartButton}
    >
      ${StartIcon}
    </${HoverTactileButton}>
    <${HoverTactileButton} onClick=${() => onCommandChange("stop")}
      disabled=${disableStopButton}
    >
      ${StopIcon}
    </${HoverTactileButton}>
    <${HoverTactileButton} onClick=${() => onCommandChange("restart")}
      disabled=${disableRestartButton}
    >
      ${RestartIcon}
    </${HoverTactileButton}>
  `;
};

const ServiceDetails = () => {
  const service = useSelectedService();
  if (!service) return null;

  const { serviceName, status, setShellRunOpts } = service;
  let state = State.Unknown;
  if (status) {
    state = statusToState(status);
  }

  const disableLogsButton = state !== State.Running;
  const disableEditButton = state !== State.Running;

  const isSystemService = serviceName === "catch";

  const controls = html`
    <${ServiceControls}
      disabled=${isSystemService}
      state=${state}
      onCommandChange=${(command) =>
        setShellRunOpts({ service: serviceName, command })}
    />
  `;

  return html`
    <div className="flex flex-col flex-grow">
      <div className="flex justify-between items-center">
        <div className="flex space-x-2">
          ${controls}
        </div>
        <h2 className="text-xl text-green-500">${serviceName}</h2>
        <div className="flex space-x-2">
          <${HoverTactileButton}
            onClick=${() =>
              setShellRunOpts({ service: serviceName, command: "edit" })}
            disabled=${disableEditButton}
          >
            ${GearIcon}
            Config
          </${HoverTactileButton}>
          <${HoverTactileButton}
            onClick=${() =>
              setShellRunOpts({
                service: serviceName,
                command: "edit",
                args: ["--env"],
              })}
            disabled=${disableLogsButton}
          >
            ${EditIcon}
            Edit env
          </${HoverTactileButton}>
        </div>
      </div>
      <div className="flex flex-col flex-grow mt-4 gap-y-4 mt-4">
        <${TerminalTabs} service=${service} />
      </div>
    </div>
  `;
};

export default ServiceDetails;
