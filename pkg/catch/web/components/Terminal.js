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

import React, { useEffect } from "react";
import htm from "htm";
import { useXTerm } from "react-xtermjs";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";

const html = htm.bind(React.createElement);

export default function Terminal({ onReady }) {
  const { instance, ref } = useXTerm();

  const fitAddon = new FitAddon();
  const webLinksAddon = new WebLinksAddon();
  useEffect(() => {
    if (!instance) return;

    instance.options.cursorBlink = true;
    instance.options.fontFamily = "Roboto Mono, monospace";
    instance.options.theme = {
      background: "#0F172A",
      black: "#000000",
      blue: "#bd93f9",
      brightBlack: "#4d4d4d",
      brightBlue: "#caa9fa",
      brightCyan: "#9aedfe",
      brightGreen: "#5af78e",
      brightMagenta: "#ff92d0",
      brightRed: "#ff6e67",
      brightWhite: "#e6e6e6",
      brightYellow: "#f4f99d",
      cursor: "#f8f8f2",
      cyan: "#8be9fd",
      foreground: "#f8f8f2",
      green: "#50fa7b",
      magenta: "#ff79c6",
      red: "#ff5555",
      selectionBackground: "#4b4f66",
      selectionForeground: "#f8f8f2",
      white: "#bfbfbf",
      yellow: "#f1fa8c",
    };

    setTimeout(() => {
      fitAddon.fit();
      onReady(instance);
    }, 0);
  }, [instance, fitAddon, onReady]);

  useEffect(() => {
    instance?.loadAddon(fitAddon);
    instance?.loadAddon(webLinksAddon);

    const handleResize = () => {
      if (instance) {
        const bodyHeight = document.body.clientHeight;
        const termBounds = ref.current.getBoundingClientRect();
        const termNewHeight = bodyHeight - termBounds.top;
        const rows = instance.rows;
        const rowHeight = termBounds.height / rows;
        let newRows = Math.floor(termNewHeight / rowHeight);
        if (isNaN(newRows) || newRows === Infinity) {
          console.log("newRows was NaN or Infinity");
          newRows = instance.rows;
        }
        instance.resize(instance.cols, newRows);
      }
      fitAddon.fit();
    };

    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, [ref, instance]);

  return html`<div style=${{ width: "100%", height: "100%" }} ref=${ref} />`;
}
