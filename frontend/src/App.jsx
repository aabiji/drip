import { useContext, useState } from "react";

import { ThemeContext } from "./StateProvider";
import StateProvider from "./StateProvider";

import SharePane from "./Share";
import ReceivedFiles from "./Files";
import Settings from "./Settings";

import settingsIcon from "./assets/settings.svg";

const Panes = { Share: 0, Received: 1, Settings: 2 };

export default function App() {
  const [activePane, setActivePane] = useState(Panes.Share);
  const {theme, _} = useContext(ThemeContext);

  return (
    <StateProvider>
      <div className="app-wrapper" data-theme={theme}>
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

          <button
            className={activePane == Panes.Settings ? "settings-active" : "settings"}
            onClick={() => setActivePane(Panes.Settings)}
          >
            <img src={settingsIcon} />
          </button>
        </div>

        <div className="content">
          {activePane == Panes.Share && <SharePane />}
          {activePane == Panes.Received && <ReceivedFiles />}
          {activePane == Panes.Settings && <Settings />}
        </div>
      </div>
    </StateProvider>
  );
}
