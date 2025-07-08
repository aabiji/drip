import { useState } from "react";
import { ExampleExportedFunc } from "../wailsjs/go/main/App";

import SharePane from "./Share";
import ReceivedFiles from "./Files";
import Settings from "./Settings";

import settingsIcon from "./assets/settings.svg";

const Panes = { Share: 0, Received: 1, Settings: 2 };

export default function App() {
  const [activePane, setActivePane] = useState(Panes.Share);

  return (
    <div class="app-wrapper">
      <div class="navbar">
        <div class="panes">
          <button
            onClick={() => setActivePane(Panes.Share)}
            class={activePane == Panes.Share ? "active" : ""}>
            Share
          </button>

          <button
            onClick={() => setActivePane(Panes.Received)}
            class={activePane == Panes.Received ? "active" : ""}>
            Received
          </button>
        </div>

        <button
          class={activePane == Panes.Settings ? "settings-active" : "settings"}
          onClick={() => setActivePane(Panes.Settings)}>
          <img src={settingsIcon}/>
        </button>
      </div>

      <div class="content">
        {activePane == Panes.Share && <SharePane />}
        {activePane == Panes.Received && <ReceivedFiles />}
        {activePane == Panes.Settings && <Settings />}
      </div>
    </div>
  );
}
