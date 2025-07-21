import { useState } from "react";

import ErrorTray from "./Error";
import TransferView from "./Transfer";
import SettingsView from "./Settings";

import { ReactComponent as SettingsIcon } from "./assets/settings.svg";
import { ReactComponent as BackIcon } from "./assets/back.svg";

const Views = { Transfer: 0, Authorize: 1, Received: 2, Settings: 3 };

export default function App() {
  const [view, setView] = useState(Views.Settings);

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

      {view == Views.Authorize &&
        <div className="content">
          <h2>So and so wants to send you some files</h2>
          <div className="options">
            <button className="accept">Accept</button>
            <button className="decline">Reject</button>
          </div>
        </div>
      }

      {view == Views.Received &&
        <div className="content">
          <h2>Received 10 files</h2>
          <div className="options">
            <button className="ok">Open</button>
          </div>
        </div>
      }

      {view == Views.Transfer && <TransferView />}
      {view == Views.Settings && <SettingsView />}
      <ErrorTray />
    </div>
  );
}
