import React, { useState } from "react";
import { createRoot } from "react-dom/client";

import State from "./State";
import ErrorTray from "./Error";
import TransferView from "./Transfer";
import SettingsView from "./Settings";

import { Settings, ArrowLeft } from "feather-icons-react";

import "./style.css";

function App() {
  const Views = { Transfer: 0, Settings: 3 };
  const [view, setView] = useState(Views.Transfer);

  return (
    <div className="app-wrapper">
      {view == Views.Transfer && (
        <button
          onClick={() => setView(Views.Settings)}
          className="fixed-button transparent-button"
        >
          <Settings className="icon" />
        </button>
      )}

      {view == Views.Settings && (
        <button onClick={() => setView(Views.Transfer)}
          className="fixed-button transparent-button">
          <ArrowLeft className="icon" />
        </button>
      )}

      {view == Views.Transfer && <TransferView />}
      {view == Views.Settings && <SettingsView />}
      <ErrorTray />
    </div>
  );
}

const container = document.getElementById("root");
const root = createRoot(container);

root.render(
  <React.StrictMode>
    <State>
      <App />
    </State>
  </React.StrictMode>,
);
