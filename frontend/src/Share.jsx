import { useContext, useEffect, useState } from "react";
import { StartFileTransfer, SendFileChunk } from "../wailsjs/go/main/App";
import { PeersContext } from "./StateProvider";

import { ReactComponent as UploadIcon } from "./assets/upload.svg";

export default function SharePane() {
  const peers = useContext(PeersContext);
  const [canSend, setCanSend] = useState(true);

  // Fetch list of peers from the backend
  useEffect(() => { setCanSend(peers && peers.length > 0) }, [peers]);

  const [selectedPeers, setSelectedPeers] = useState([]);
  const [selectedFiles, setSelectedFiles] = useState([]);

  const selectPeer = (event, name) => {
    setSelectedPeers((prev) => {
      const list = event.target.checked
        ? [...prev, name]
        : prev.filter((peer) => peer != name);
      return list;
    });
  };

  const selectFiles = (event) => {
    setSelectedFiles((prev) => {
      const files = Array.from(event.target.files);
      const existing = new Set(prev.map((f) => `${f.name}-${f.size}`));
      const unique = files.filter((f) => !existing.has(`${f.name}-${f.size}`));
      return [...prev, ...unique];
    });
  }

  const removeFile = (name) =>
    setSelectedFiles((prev) => prev.filter((f) => f.name != name));

  const streamFile = async (file) => {
    const chunkSize = 5 * 1024 * 1024; // 5 megabytes
    const numChunks = Math.ceil(file.size / chunkSize);
    const info = { recipients: selectedPeers, name: file.name, size: file.size, numChunks };

    const ok = StartFileTransfer(info);
    if (!ok) {
      console.log("tell the user the error!");
      return;
    }

    for (let i = 0; i < numChunks; i++) {
      const slice = file.slice(i * chunkSize, (i + 1) * chunkSize);
      const chunkData = new Uint8Array(await slice.arrayBuffer());
      const chunk = { data: Array.from(chunkData), chunkIndex: i };
      const ok = await SendFileChunk(chunk);
      if (!ok) {
        console.log("tell the user the error!"); // TODO: stop!
        return;
      }
    }

    // TODO: redirect to upload page
  };

  const sendFiles = async () => {
    if (selectedFiles.length == 0 || selectedPeers.length == 0) return;
    try {
      for (const file of selectedFiles) {
        await streamFile(file);
      }
    } catch (error) {
      console.log("tell the user the error!");
    }
  };

  return (
    <div className="inner-content">
      <div className="upper-container">
        <h3> Send to </h3>
        {canSend ? (
          <div className="peers-container">
            {peers.map((name, index) => (
              <div className="peer-entry" key={index}>
                <label className="custom-checkbox">
                  <input
                    type="checkbox" className="checkbox"
                    onChange={(event) => selectPeer(event, name)} />
                  <span className="fake-checkbox"></span>
                </label>
                <p>{name}</p>
              </div>
            ))}
          </div>
        ) : (
          <p> There are no devices around to connect to </p>
        )}
      </div>

      <div className="upload-container">
        <div className="file-input-container">
          <label className={canSend ? "file-label" : "file-label disabled"}>
            <UploadIcon className="upload-icon" />
            <p>Drag and drop or choose files</p>
            <input
              type="file" disabled={!canSend}
              onChange={(event) => selectFiles(event)} />
          </label>
        </div>
        <div className="file-selection-container">
          {selectedFiles.map((file, index) => (
            <div className="file-selection" key={index}>
              <p> {file.name} </p>
              <button onClick={() => removeFile(file.name)}> x </button>
            </div>
          ))}
        </div>
        <button
          className="send-button" disabled={!canSend}
          onClick={sendFiles}>Send</button>
      </div>
    </div>
  );
}
