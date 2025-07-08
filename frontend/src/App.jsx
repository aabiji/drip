import { useState } from "react";
import { ExampleExportedFunc } from "../wailsjs/go/main/App";

import uploadIcon from "./assets/upload.svg";
import fileIcon from "./assets/file.svg";
import folderIcon from "./assets/folder.svg";
import sunIcon from "./assets/sun.svg";
import moonIcon from "./assets/moon.svg";
import settingsIcon from "./assets/settings.svg";

const Panes = { Share: 0, Received: 1, Settings: 2 };

function SharePane() {
  const deviceNames = [
    "Device A", "Device B", "Device C", "Device D",
  ];

  return (
    <div class="inner-content">
      <div class="devices-container">
        {deviceNames.map((name) => (
          <div class="device-entry">
            <label class="custom-checkbox">
              <input type="checkbox" class="checkbox" />
              <span class="fake-checkbox"></span>
            </label>
            <p>{name}</p>
          </div>
        ))}
      </div>

      <div class="upload-container">
        <div class="file-input-container">
          <label class="file-label">
            <img src={uploadIcon} class="upload-icon" alt="Upload" />
            <p>Drag and drop or choose files</p>
            <input type="file" />
          </label>
        </div>
        <button class="send-button">Send</button>
      </div>
    </div>
  );
}

function ReceivedPane() {
  const transfers = [
    { path: "/path/to/fileA",   sentFrom: "Device A", folder: false },
    { path: "/path/to/fileB",   sentFrom: "Device B", folder: false },
    { path: "/path/to/folderA", sentFrom: "Device C", folder: true },
  ];

  return (
    <div class="transfer-grid">

      {transfers.map((transfer) => (
        <div class="transfer-card">
          <img src={transfer.folder ? folderIcon : fileIcon} class="banner" />
          <div class="info">
            <b> {transfer.path} </b>
            <p> Sent from {transfer.sentFrom} </p>
          </div>
        </div>
      ))}

    </div>
  );
}

function SettingsPane() {
  const startYear = 2025;
  const currentYear = new Date().getFullYear();
  const copyright = startYear == currentYear ? `${startYear}`: `${startYear}-${currentYear}`;

  const [downloadPath, setDownloadPath] = useState("~/Downloads/");

  const [isLightMode, setIsLightMode] = useState(true);

  return (
    <div class="inner-content">
      <div class="row">
        <p class="input-label">Toggle theme</p>
        <button class="icon-button" onClick={() => setIsLightMode(!isLightMode)}>
          <img src={isLightMode ? moonIcon : sunIcon} alt="Toggle Theme" />
        </button>
      </div>

      <div className="row">
        <p className="input-label">Download folder</p>
        <label className="folder-label">
          <p className="path">{downloadPath}</p>
          <input type="file" webkitdirectory className="folder-path-input" />
        </label>
      </div>

      <p class="copyright">© Abigail Adegbiji @aabiji, {copyright}</p>
    </div>
  );
}

export default function App() {
  const [activePane, setActivePane] = useState(Panes.Settings);

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

        <button class="settings"
          onClick={() => setActivePane(Panes.Settings)}>
          <img src={settingsIcon}/>
        </button>
      </div>

      <div class="content">
        {activePane == Panes.Share && <SharePane />}
        {activePane == Panes.Received && <ReceivedPane />}
        {activePane == Panes.Settings && <SettingsPane />}
      </div>
    </div>
  );
}
