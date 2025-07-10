import { useState } from "react";

import SharePane from "./Share";
import ReceivedFiles from "./Files";
import Settings from "./Settings";

import { ReactComponent as SettingsIcon } from "./assets/settings.svg";

const Panes = { Share: 0, Received: 1, Settings: 2 };

export default function App() {
  const [activePane, setActivePane] = useState(Panes.Share);

  return (
    <div className="app-wrapper">
      <div className="navbar">
        <div className="panes">
          <button
            onClick={() => setActivePane(Panes.Share)}
            className={activePane == Panes.Share ? "active" : ""}
          >
            Share
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
        {activePane == Panes.Share && <SharePane />}
        {activePane == Panes.Received && <ReceivedFiles />}
        {activePane == Panes.Settings && <Settings />}
      </div>
    </div>
  );
}
