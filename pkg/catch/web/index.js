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

import React, { useState } from "react";
import ReactDOM from "react-dom";
import clsx from "clsx";
import htm from "htm";

import ServiceList from "./components/ServiceList.js";
import ServiceDetails from "./components/ServiceDetails.js";
import { ServiceProvider } from "./context/ServiceContext.js";

const html = htm.bind(React.createElement);

const Logo = ({ className }) => {
  return html`
    <div
      className=${clsx(
        "flex flex-row items-center gap-2 select-none text-green-500",
        className
      )}
    >
      <img src="/img/soth.svg" width="64" height="64" />
      <span className="text-3xl">❯</span>
      <span className="text-3xl">yeet</span>
    </div>
  `;
};

const LeftPanel = () => {
  return html`
    <div className="md:fixed md:inset-y-0 md:z-50 md:flex md:w-72 md:flex-col">
      <div
        className="flex grow flex-col gap-y-5 overflow-y-auto bg-gray-900 px-6 pb-4 bg-slate-800"
      >
        <div className="flex mt-4 h-16 shrink-0 items-center">
          <${Logo} className="m-4 -ml-2" />
        </div>
        <nav className="flex flex-1 flex-col">
          <ul role="list" className="flex flex-1 flex-col gap-y-7">
            <${ServiceList} />
          </ul>
        </nav>
      </div>
    </div>
  `;
};

const RightPanel = () => {
  return html`
    <div className="flex md:pl-72 flex-grow pt-10">
      <main className="flex-grow flex flex-col">
        <div className="flex px-4 sm:px-6 md:px-8 flex-grow">
          <${ServiceDetails} />
        </div>
      </main>
    </div>
  `;
};

const App = () => {
  return html`
  <${ServiceProvider}>
    <div className="min-h-screen flex flex-col">
      <${LeftPanel} />
      <${RightPanel} />
    </div>
  </${ServiceProvider}>
  `;
};

const root = ReactDOM.createRoot(document.getElementById("root"));
root.render(html`<${App} foo=${"bar"} />`);
