import { useContext, useEffect, useState, useRef } from "react";

import { PeersContext } from "./State";

import { Upload, X } from "feather-icons-react";
import { PuffLoader } from "react-spinners";

export function FileEntry({ name, progress, onClick, recipient, error }) {
  const barElement = useRef();
  const [msg, setMsg] = useState(
    recipient !== undefined ? `Sending ${name} to ${recipient}` : name,
  );

  useEffect(() => {
    if (error && recipient !== undefined) {
      progress = undefined;
      setMsg(`Failed to send ${name} to ${recipient}`);
    }

    if (progress !== undefined && progress > 0) {
      barElement.current.style.width = `${Math.min(progress, 1.0) * 100}%`;
    } else if (barElement.current) {
      barElement.current.style.width = "0px";
    }
  }, [progress, error]);

  return (
    <div className={error ? "file-entry error" : "file-entry"}>
      <p>{msg}</p>
      {onClick !== undefined &&
        <button className="transparent-button" onClick={onClick}>
          <X className="icon" />
        </button>}
      {progress !== undefined && (
        <div className="progress-bar" ref={barElement}></div>
      )}
    </div>
  );
}

export default function TransferSelection({
  setSending,
  selectedPeers,
  setSelectedPeers,
  selectedFiles,
  setSelectedFiles,
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
      if (await canReadFile(file)) validFiles.push(file);
    }

    setSelectedFiles((prev) => [...prev, ...validFiles]);
  };

  const removeFile = (name) =>
    setSelectedFiles((prev) => prev.filter((f) => f.name != name));

  const dragOverHandler = (event) => {
    event.preventDefault();
  };

  const traverseEntry = (entry) => {
    return new Promise((resolve) => {
      if (entry.isFile) {
        entry.file((file) => resolve([file]));
      } else if (entry.isDirectory) {
        const reader = entry.createReader();
        const entries = [];

        const readEntries = () => {
          reader.readEntries(async (batch) => {
            if (batch.length === 0) {
              const promises = entries.map(traverseEntry);
              const files = await Promise.all(promises);
              resolve(files.flat());
            } else {
              entries.push(...batch);
              readEntries();
            }
          });
        };
        readEntries();
      }
    });
  };

  const dropHandler = async (event) => {
    event.preventDefault();

    const items = event.dataTransfer.items;
    const filePromises = [];

    if (items) {
      for (const item of items) {
        if (item.kind === "file") {
          const entry = item.webkitGetAsEntry();
          if (entry) {
            filePromises.push(traverseEntry(entry));
          }
        }
      }

      const files = (await Promise.all(filePromises)).flat();
      addNonDuplicateFiles(files);
    }
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
                    type="checkbox"
                    className="checkbox"
                    checked={selectedPeers.includes(name)}
                    onChange={(event) => selectPeer(event, name)}
                  />
                  <span className="fake-checkbox"></span>
                </label>
                <p>{name}</p>
              </div>
            ))}
          </div>
        ) : (
          <div className="loader-row">
            <PuffLoader color="#007aff" loading={true} size={35} />
            <p> Searching for peers </p>
          </div>
        )}
      </div>

      <div className="upload-container">
        <div
          className="file-input-container"
          onDrop={(event) => dropHandler(event)}
          onDragOver={(event) => dragOverHandler(event)}
        >
          <label className="file-label">
            <Upload className="upload-icon" />
            <p>Choose or drag and drop files</p>
            <input
              type="file"
              multiple
              webkitdirectory="true"
              onChange={(event) =>
                addNonDuplicateFiles(Array.from(event.target.files))
              }
            />
          </label>
        </div>
        <div className="file-selection-container">
          {selectedFiles.map((file, index) => (
            <FileEntry
              key={index}
              name={file.name}
              onClick={() => removeFile(file.name)}
            />
          ))}
        </div>
        <button className="send-button"
          disabled={!canSend} onClick={() => setSending(true)}>
          Send files
        </button>
      </div>
    </div>
  );
}
