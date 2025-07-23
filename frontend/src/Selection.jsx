import { useContext, useEffect, useState, useRef } from "react";
import { Upload } from "feather-icons-react";

import { PeersContext } from "./State";

export function FileEntry({ name, progress, onClick, recipient, error }) {
  const barElement = useRef();
  const [full, setFull] = useState(false);
  const [msg, setMsg] = useState(
    recipient !== undefined ? `Sending ${name} to ${recipient}` : name
  );

  useEffect(() => {
    if (error && recipient !== undefined) {
      progress = undefined;
      setMsg(`Failed to send ${name} to ${recipient}`);
    }

    if (progress !== undefined && progress > 0) {
      barElement.current.style.width = `${Math.min(progress, 1.0) * 100}%`;
      setFull(progress >= 1.0);
    } else if (barElement.current) {
      barElement.current.style.width = "0px";
    }
  }, [progress, error]);

  return (
    <div className={full ? "file-entry full" : error ? "file-entry error" : "file-entry"}>
      <div className="inner">
        <p>{msg}</p>
        {onClick !== undefined && <button onClick={onClick}>x</button>}
      </div>
      {progress !== undefined && <div className="progress-bar" ref={barElement}></div>}
    </div>
  );
}

// TODO: select all the files inside a folder
export default function TransferSelection({
  setSending, selectedPeers, setSelectedPeers,
  selectedFiles, setSelectedFiles
}) {
  const peers = useContext(PeersContext);
  const [havePeers, setHavePeers] = useState(false);
  const [canSend, setCanSend] = useState(false);

  // Fetch list of peers from the backend
  useEffect(() => {
    setHavePeers(peers && peers.length > 0);
    setCanSend(selectedFiles.length > 0 && selectedPeers.length > 0);
  }, [peers, selectedFiles, selectedPeers]);

  const selectPeer = (event, name) => {
    setSelectedPeers((prev) => {
      const list = event.target.checked
        ? [...prev, name]
        : prev.filter((peer) => peer != name);
      return list;
    });
  };

  const canReadFile = async (file) => {
    try {
      await file.slice(0, 1).arrayBuffer(); // Try to read 1 byte
      return true;
    } catch (e) {
      return false;
    }
  };

  const addNonDuplicateFiles = async (files) => {
    const existing = new Set(selectedFiles.map((f) => `${f.name}-${f.size}`));
    const unique = files.filter((f) => !existing.has(`${f.name}-${f.size}`));

    let validFiles = [];
    for (const file of unique) {
      if (await canReadFile(file))
        validFiles.push(file);
    }

    setSelectedFiles((prev) => [...prev, ...validFiles]);
  };

  const removeFile = (name) =>
    setSelectedFiles((prev) => prev.filter((f) => f.name != name));

  const dragOverHandler = (event) => { event.preventDefault(); }

  const dropHandler = (event) => {
    event.preventDefault();
    let files = [];
    if (event.dataTransfer.items) {
      files = Array.from(event.dataTransfer.items)
        .filter((item) => item.kind === "file")
        .map((item) => item.getAsFile());
    } else {
      files = event.dataTransfer.files;
    }
    addNonDuplicateFiles(files);
  };

  return (
    <div className="content">
      <div className="upper-container">
        {havePeers && peers ? (
          <div className="peers-container">
            {peers.map((name, index) => (
              <div className="peer-entry" key={index}>
                <label className="custom-checkbox">
                  <input
                    type="checkbox" className="checkbox"
                    checked={selectedPeers.includes(name)}
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
        <div
          className="file-input-container"
          onDrop={(event) => dropHandler(event)}
          onDragOver={(event) => dragOverHandler(event)}>
          <label className="file-label">
            <Upload className="upload-icon" />
            <p>Choose or drag and drop files</p>
            <input type="file" multiple webkitdirectory
                onChange={(event) => addNonDuplicateFiles(Array.from(event.target.files))} />
          </label>
        </div>
        <div className="file-selection-container">
          {selectedFiles.map((file, index) => (
            <FileEntry key={index} name={file.name} onClick={() => removeFile(file.name)} />
          ))}
        </div>
        <button
          className="send-button" disabled={!canSend}
          onClick={() => setSending(true)}>Send files</button>
      </div>
    </div>
  );
}