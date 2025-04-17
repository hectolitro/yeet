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

import React, { useCallback, useEffect, useRef, useState } from "react";
import htm from "htm";
import { AttachAddon } from "@xterm/addon-attach";

import Terminal from "./Terminal.js";

const html = htm.bind(React.createElement);

const ServiceTerminal = ({ runOpts, setConnected = () => {}, ...props }) => {
  const [terminal, setTerminal] = useState(null);

  const onTerminalReady = useCallback(
    (terminal) => {
      setTerminal(terminal);
    },
    [setTerminal]
  );

  const wsRef = useRef(null);
  const connectWebSocket = useCallback(
    (url) => {
      if (wsRef.current) {
        wsRef.current.close();
      }
      wsRef.current = new WebSocket(url);
      const ws = wsRef.current;

      ws.onopen = () => {
        terminal.clear();
        terminal.loadAddon(new AttachAddon(ws));
        setConnected(true);
      };

      ws.onclose = () => {
        terminal.writeln("");
        terminal.writeln("Connection closed");
        setConnected(false);
      };

      ws.onerror = (error) => {
        terminal.writeln(
          `Terminal WebSocket error: ${
            error.message || "An unknown error occurred"
          }`
        );
        setConnected(false);
      };

      terminal.onResize(({ cols, rows }) => {
        // Send resize control sequence to server
        const resizeMessage = new TextEncoder().encode(`[8;${rows};${cols}t`);
        const messageArray = new Uint8Array([0x01, ...resizeMessage]);
        ws.send(messageArray);
      });
    },
    [terminal, runOpts, setConnected]
  );

  useEffect(() => {
    if (!terminal) return;
    if (!runOpts) return;

    const { service, command, args } = runOpts;

    const url = new URL("/api/v0/run-command", window.location.origin);
    url.protocol = window.location.protocol === "http:" ? "ws:" : "wss:";
    url.searchParams.set("rows", terminal.rows);
    url.searchParams.set("cols", terminal.cols);
    url.searchParams.set("service", service);
    url.searchParams.set("command", command);
    args &&
      args.forEach((arg) => {
        url.searchParams.append("args", arg);
      });

    connectWebSocket(url);

    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
      terminal.clear();
    };
  }, [terminal, runOpts, connectWebSocket]);

  return html` <${Terminal} onReady=${onTerminalReady} ...${props} /> `;
};

export default ServiceTerminal;
