import { useState } from "react";
import { ExampleExportedFunc } from "../wailsjs/go/main/App";

import uploadIcon from "./assets/upload.svg";
import fileIcon from "./assets/file.svg";
import folderIcon from "./assets/folder.svg";

const Panes = { Share: 0, Received: 1, Settings: 2 };

function SharePane() {
  const deviceNames = [
    "Device A", "Device B", "Device C", "Device D",
  ];

  return (
    <div class="share-content">
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
    <div>

      {transfers.map((transfer) => (
        <div class="transfer-card">
          <img src={transfer.folder ? folderIcon : fileIcon} class="banner" />
          <div class="info">
            <p> {transfer.sentFrom} </p>
            <p> {transfer.path} </p>
          </div>
        </div>
      ))}

    </div>
  );
}

function SettingsPane() {
  return (
    <div>
      <button> Toggle theme </button>
      <label> Download folder </label>
      <input type="file" webkitdirectory id="folderInput" />
    </div>
  );
}

export default function App() {
  const [activePane, setActivePane] = useState(Panes.Received);

  return (
    <div class="app-wrapper">
      <div class="navbar">
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

        <button
          onClick={() => setActivePane(Panes.Settings)}
          class={activePane == Panes.Settings ? "active" : ""}>
          Settings
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
