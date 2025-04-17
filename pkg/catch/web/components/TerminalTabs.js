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

import React, { useEffect, useMemo, useRef, useState } from "react";
import clsx from "clsx";
import htm from "htm";

import { useShellRunOpts } from "../context/ServiceContext.js";
import { State, statusToState } from "../lib/status.js";
import ServiceTerminal from "./ServiceTerminal.js";
import { LogsTab, ShellTab } from "./Tabs.js";

const html = htm.bind(React.createElement);

const tabs = {
  Logs: 0,
  Shell: 1,
};

const TerminalTabs = ({ service }) => {
  if (!service) return null;

  const [selectedTab, setSelectedTab] = useState(tabs.Logs);

  const { serviceName, setShellRunOpts, status } = service;
  let state = State.Unknown;
  if (status) {
    state = statusToState(status);
  }

  const logsOpts = useMemo(
    () => ({
      service: serviceName,
      command: "logs",
      args: ["-f", "-n", "1000"],
    }),
    [serviceName]
  );

  const logsTerminal = useMemo(() => {
    return html`<${ServiceTerminal}
      runOpts=${logsOpts}
      reconnectOnClose=${true}
    /> `;
  }, [logsOpts, state === State.Stopped]); /* state is used to auto-reconnect */

  const shellRunOpts = useShellRunOpts();
  useEffect(() => {
    if (shellRunOpts) {
      setSelectedTab(tabs.Shell);
    } else {
      setSelectedTab(tabs.Logs);
    }
  }, [shellRunOpts]);

  useEffect(() => {
    setShellRunOpts(null);
    setSelectedTab(tabs.Logs);
  }, [serviceName]);

  const shellTerminal = useMemo(() => {
    if (!shellRunOpts) return null;

    return html`<${ServiceTerminal} runOpts=${shellRunOpts} /> `;
  }, [shellRunOpts]);

  return html`<div className="flex flex-col flex-grow">
    <div className="flex space-x-2 mt-1 mb-4">
      <${LogsTab}
        className="w-60"
        selected=${selectedTab === tabs.Logs}
        onClick=${() => setSelectedTab(tabs.Logs)}
      />
      ${shellRunOpts &&
      html`<${ShellTab}
        className="w-60"
        selected=${selectedTab === tabs.Shell}
        onClick=${() => setSelectedTab(tabs.Shell)}
        onClose=${() => setShellRunOpts(null)}
      />`}
    </div>
    <div
      className=${clsx(["flex-grow", selectedTab !== tabs.Logs && "hidden"])}
    >
      ${logsTerminal}
    </div>
    <div
      className=${clsx(["flex-grow", selectedTab !== tabs.Shell && "hidden"])}
    >
      ${shellTerminal}
    </div>
  </div>`;
};

export default TerminalTabs;
