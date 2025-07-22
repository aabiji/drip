import React, { useState } from "react";
import { createRoot } from "react-dom/client";

import State from "./State";
import ErrorTray from "./Error";
import TransferView from "./Transfer";
import SettingsView from "./Settings";

import { ReactComponent as SettingsIcon } from "./assets/settings.svg";
import { ReactComponent as BackIcon } from "./assets/back.svg";
import "./style.css";

function App() {
  const Views = { Transfer: 0,  Settings: 3 };
  const [view, setView] = useState(Views.Transfer);

  return (
    <div className="app-wrapper">
      {view == Views.Transfer &&
        <button onClick={() => setView(Views.Settings)} className="settings-button">
          <SettingsIcon className="settings-icon" />
        </button>
      }

      {view == Views.Settings &&
        <button onClick={() => setView(Views.Transfer)} className="back-button">
          <BackIcon class="settings-icon" />
        </button>
      }

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
