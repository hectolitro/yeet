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
  base: "group flex select-none cursor-pointer font-mono text-sm text-green-600",
  container: "relative rounded border",
  inner: "relative flex items-center gap-x-2 rounded border px-4 py-1.5 -top-1",
  active: "group-active:-top-0.5",
  hover: {
    container:
      "border-transparent group-hover:border-slate-700 group-hover:bg-slate-600",
    inner:
      "border-transparent group-hover:border-slate-500 group-hover:bg-slate-800",
  },
  tactile: {
    container: "border-slate-700 bg-slate-600",
    inner: "border-slate-500 bg-slate-800",
  },
  disabled: "cursor-auto border-transparent opacity-75",
  enabled: "hover:text-green-500",
};

const HoverTactileButton = ({
  children,
  onClick,
  className,
  disabled,
  ...props
}) => {
  return html` <button
    class=${clsx(
      styles.base,
      className,
      disabled ? styles.disabled : styles.enabled
    )}
    onClick=${disabled ? undefined : onClick}
    disabled=${disabled}
    ...${props}
  >
    <span
      class=${clsx(
        styles.container,
        !disabled && styles.hover.container,
        disabled && styles.disabled
      )}
    >
      <span
        class=${clsx(
          styles.inner,
          !disabled && styles.active,
          !disabled && styles.hover.inner,
          disabled && styles.disabled
        )}
        >${children}</span
      >
    </span>
  </button>`;
};

const TactileButton = ({
  children,
  onClick,
  className,
  disabled,
  ...props
}) => {
  return html` <button
    class=${clsx(styles.base, className, disabled && styles.disabled)}
    onClick=${onClick}
    disabled=${disabled}
    ...${props}
  >
    <span class=${clsx(styles.container, styles.tactile.container)}>
      <span class=${clsx(styles.inner, styles.tactile.inner)}>${children}</span>
    </span>
  </button>`;
};

export { HoverTactileButton, TactileButton };
