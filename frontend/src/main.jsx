import React from "react";
import { createRoot } from "react-dom/client";

import App from "./App";
import State from "./State";

import "./style.css";

const container = document.getElementById("root");

const root = createRoot(container);

root.render(
  <React.StrictMode>
    <State>
      <App />
    </State>
  </React.StrictMode>,
);
