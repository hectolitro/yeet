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

import React, {
  createContext,
  useContext,
  useEffect,
  useReducer,
  useRef,
} from "react";
import htm from "htm";

const html = htm.bind(React.createElement);

export const ServiceContext = createContext(null);

const initialState = {
  selectedService: "catch",
  services: { catch: { serviceName: "catch" } },
  shellRunOpts: null,
  lastHeartbeat: null,
};

// ActionTypes are internal or external event types
export const ActionTypes = {
  INITIALIZE_SERVICES: "InitializeServices",
  SET_SELECTED_SERVICE: "SetSelectedService",
  SET_SHELL_RUN_OPTS: "SetShellRunOpts",
  HEARTBEAT: "Heartbeat",
  SERVICE_STATUS_CHANGED: "ServiceStatusChanged",
  SERVICE_CREATED: "ServiceCreated",
  SERVICE_DELETED: "ServiceDeleted",
  SERVICE_CONFIG_CHANGED: "ServiceConfigChanged",
};

function serviceReducer(state, action) {
  switch (action.type) {
    case ActionTypes.INITIALIZE_SERVICES:
      return {
        ...state,
        services: action.payload.reduce((acc, data) => {
          acc[data.serviceName] = {
            serviceName: data.serviceName,
            status: data,
            config: null,
          };
          return acc;
        }, {}),
      };
    case ActionTypes.SET_SELECTED_SERVICE:
      return {
        ...state,
        selectedService: action.payload,
      };
    case ActionTypes.SET_SHELL_RUN_OPTS:
      return { ...state, shellRunOpts: action.payload };
    case ActionTypes.HEARTBEAT:
      return { ...state, lastHeartbeat: action.payload };
    case ActionTypes.SERVICE_STATUS_CHANGED:
      return {
        ...state,
        services: {
          ...state.services,
          [action.payload.serviceName]: {
            ...state.services[action.payload.serviceName],
            status: action.payload.data,
          },
        },
      };
    case ActionTypes.SERVICE_CREATED:
    case ActionTypes.SERVICE_CONFIG_CHANGED:
      return {
        ...state,
        services: {
          ...state.services,
          [action.payload.serviceName]: {
            ...state.services[action.payload.serviceName],
            serviceName: action.payload.serviceName,
            config: action.payload.data,
          },
        },
      };
    case ActionTypes.SERVICE_DELETED:
      const { [action.payload.serviceName]: _, ...remainingServices } =
        state.services;
      const selectedService =
        state.selectedService === action.payload.serviceName
          ? null
          : state.selectedService;
      return {
        ...state,
        services: remainingServices,
        selectedService,
      };
    default:
      console.error("Unhandled action type", action.type);
      return state;
  }
}

export const ServiceProvider = ({ children }) => {
  const [state, dispatch] = useReducer(serviceReducer, initialState);

  // Ensure a single WebSocket instance is used
  const wsRef = useRef(null);
  const reconnectAttemptsRef = useRef(0);

  useEffect(() => {
    const websocketUrl = `${
      window.location.protocol === "http:" ? "ws:" : "wss:"
    }//${window.location.host}/api/v0/events`;

    const connectWebSocket = () => {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }

      wsRef.current = new WebSocket(websocketUrl);
      const ws = wsRef.current;

      ws.onopen = () => {
        console.log("Events WebSocket connected");
        reconnectAttemptsRef.current = 0; // Reset attempts on successful connection
      };

      ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (Object.values(ActionTypes).includes(data.type)) {
          dispatch({ type: data.type, payload: data });
        } else {
          console.warn(`Unhandled event type: ${data.type}`);
        }
      };

      const handleReconnect = () => {
        console.log("Events WebSocket disconnected, reconnecting...");
        const backoffTime = Math.min(
          Math.pow(2, reconnectAttemptsRef.current) * 1000,
          30000
        ); // Exponential backoff to 30s max
        setTimeout(() => {
          console.log(
            `Reconnecting attempt ${reconnectAttemptsRef.current + 1}`
          );
          reconnectAttemptsRef.current += 1;

          connectWebSocket();
        }, backoffTime);
      };

      ws.onclose = handleReconnect;
      ws.onerror = (error) => {
        console.error(
          `Events WebSocket error: ${
            error.message || "An unknown error occurred"
          }`
        );
        if (ws) {
          ws.close();
        }
        handleReconnect();
      };
    };

    connectWebSocket();

    // Fetch initial state
    const url = new URL("/api/v0/run-command", window.location.origin);
    url.searchParams.set("command", "status");
    url.searchParams.set("service", "sys");
    url.searchParams.set("tty", "false");
    url.searchParams.set("args", "--format=json");
    fetch(url)
      .then((response) => response.json())
      .then((services) => {
        dispatch({ type: ActionTypes.INITIALIZE_SERVICES, payload: services });
      });

    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, []);

  return html`
    <${ServiceContext.Provider} value=${{ state, dispatch }}>
      ${children}
    </${ServiceContext.Provider}>
  `;
};

export const useService = (serviceId) => {
  const context = useContext(ServiceContext);
  if (!context) {
    throw new Error("useService must be used within a ServiceProvider");
  }
  const { state, dispatch } = context;

  const setSelectedService = () => {
    dispatch({ type: ActionTypes.SET_SELECTED_SERVICE, payload: serviceId });
  };

  const setShellRunOpts = (runOpts) => {
    dispatch({ type: ActionTypes.SET_SHELL_RUN_OPTS, payload: runOpts });
  };

  return {
    ...state.services[serviceId],
    setSelectedService,
    setShellRunOpts,
  };
};

export const useSelectedService = () => {
  const context = useContext(ServiceContext);
  if (!context) {
    throw new Error("useSelectedService must be used within a ServiceProvider");
  }
  const { state, dispatch } = context;
  const selectedServiceId = state.selectedService;

  if (!selectedServiceId) {
    return null;
  }

  const setShellRunOpts = (runOpts) => {
    dispatch({ type: ActionTypes.SET_SHELL_RUN_OPTS, payload: runOpts });
  };

  return {
    ...state.services[selectedServiceId],
    setShellRunOpts,
  };
};

export const useLastHeartbeat = () => {
  const context = useContext(ServiceContext);
  if (!context) {
    throw new Error("useLastHeartbeat must be used within a ServiceProvider");
  }
  const { state } = context;
  return state.lastHeartbeat;
};

export const useShellRunOpts = () => {
  const context = useContext(ServiceContext);
  if (!context) {
    throw new Error("useShellRunOpts must be used within a ServiceProvider");
  }
  const { state } = context;
  return state.shellRunOpts;
};
