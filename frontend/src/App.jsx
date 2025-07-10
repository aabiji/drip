import { useState } from "react";

import TransferPane from "./Transfer";
import ReceivedFiles from "./Files";
import Settings from "./Settings";

import { ReactComponent as SettingsIcon } from "./assets/settings.svg";

const Panes = { Transfer: 0, Received: 1, Settings: 2 };

export default function App() {
  const [activePane, setActivePane] = useState(Panes.Transfer);

  return (
    <div className="app-wrapper">
      <div className="navbar">
        <div className="panes">
          <button
            onClick={() => setActivePane(Panes.Transfer)}
            className={activePane == Panes.Transfer ? "active" : ""}
          >
            Transfer
          </button>

          <button
            onClick={() => setActivePane(Panes.Received)}
            className={activePane == Panes.Received ? "active" : ""}
          >
            Received
          </button>
        </div>

        <button onClick={() => setActivePane(Panes.Settings)}>
          <SettingsIcon
            className={activePane == Panes.Settings ? "settings-active-icon" : "settings-icon"} />
        </button>
      </div>

      <div className="content">
        {activePane == Panes.Transfer && <TransferPane />}
        {activePane == Panes.Received && <ReceivedFiles />}
        {activePane == Panes.Settings && <Settings />}
      </div>
    </div>
  );
}
