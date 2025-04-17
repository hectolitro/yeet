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
import clsx from "clsx";
import htm from "htm";

const html = htm.bind(React.createElement);

const styles = {
  tab: `
    flex items-center rounded py-1.5 pl-3 text-left
    border border-transparent border-slate-800
    hover:bg-slate-800 text-sm active:translate-y-0.5
  `,
  selected: "bg-slate-800 !border-slate-600 text-green-500 font-semibold",
  icon: "mr-1 size-5",
  rightIcon: "ml-auto p-1.5 mr-1.5 rounded",
  rightIconHover: "hover:bg-slate-900",
  svg: "size-4",
};

const LogsIcon = html`
  <svg
    xmlns="http://www.w3.org/2000/svg"
    fill="none"
    viewBox="0 0 24 24"
    stroke-width="1.5"
    stroke="currentColor"
    className=${styles.icon}
  >
    <path
      stroke-linecap="round"
      stroke-linejoin="round"
      d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25H12"
    />
  </svg>
`;

const ShellIcon = html`
  <svg
    xmlns="http://www.w3.org/2000/svg"
    fill="none"
    viewBox="0 0 24 24"
    stroke-width="1.5"
    stroke="currentColor"
    className=${styles.icon}
  >
    <path
      stroke-linecap="round"
      stroke-linejoin="round"
      d="m6.75 7.5 3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0 0 21 18V6a2.25 2.25 0 0 0-2.25-2.25H5.25A2.25 2.25 0 0 0 3 6v12a2.25 2.25 0 0 0 2.25 2.25Z"
    />
  </svg>
`;

const LockIcon = html`
  <svg
    xmlns="http://www.w3.org/2000/svg"
    fill="none"
    viewBox="0 0 24 24"
    stroke-width="1.5"
    stroke="currentColor"
    className=${styles.svg}
  >
    <path
      stroke-linecap="round"
      stroke-linejoin="round"
      d="M16.5 10.5V6.75a4.5 4.5 0 1 0-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H6.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z"
    />
  </svg>
`;

const CloseIcon = html`
  <svg
    xmlns="http://www.w3.org/2000/svg"
    fill="none"
    viewBox="0 0 24 24"
    stroke-width="1.5"
    stroke="currentColor"
    className=${styles.svg}
  >
    <path
      stroke-linecap="round"
      stroke-linejoin="round"
      d="M6 18 18 6M6 6l12 12"
    />
  </svg>
`;

const TabComponent = ({
  icon,
  label,
  rightIcon,
  selected,
  className,
  onClose,
  ...props
}) => {
  return html`
    <button
      className=${clsx(styles.tab, selected && styles.selected, className)}
      ...${props}
    >
      ${icon} ${label}
      <span
        className=${clsx(styles.rightIcon, onClose && styles.rightIconHover)}
        onClick=${onClose}
      >
        ${rightIcon}
      </span>
    </button>
  `;
};

export const LogsTab = ({ ...props }) => {
  return html`
    <${TabComponent}
      icon=${LogsIcon}
      label="Logs"
      rightIcon=${LockIcon}
      ...${props}
    />
  `;
};

export const ShellTab = ({ onClose, ...props }) => {
  return html`
    <${TabComponent}
      icon=${ShellIcon}
      label="Shell"
      rightIcon=${CloseIcon}
      onClose=${onClose}
      ...${props}
    />
  `;
};
