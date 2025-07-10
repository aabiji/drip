import { useContext, useEffect, useState, useRef } from "react";
import { StartFileTransfer, SendFileChunk } from "../wailsjs/go/main/App";
import { PeersContext } from "./StateProvider";

import { ReactComponent as UploadIcon } from "./assets/upload.svg";
import { ReactComponent as RetryIcon } from "./assets/retry.svg";

function FileEntry({ name, progress, onClick }) {
  const barElement = useRef();
  const [full, setFull] = useState(false);

  useEffect(() => {
    if (progress !== undefined) {
      const total = barElement.current.parentElement.offsetWidth - 2;
      barElement.current.style.width = `${progress * total}px`;
      setFull(progress >= 1.0);
    }
  }, []);

  return (
    <div className={!full ? "file-entry": "file-entry full"}>
      <div className="inner">
        <p>{name}</p>
        {onClick !== undefined  && <button onClick={onClick}>x</button>}
      </div>
      {progress !== undefined && <div className="progress-bar" ref={barElement}></div>}
    </div>
  );
}

function FileAndPeerSelection({
  peers, startSending, setSelectedPeers,
  selectedFiles, setSelectedFiles
}) {
  const [canSend, setCanSend] = useState(true);

  // Fetch list of peers from the backend
  useEffect(() => { setCanSend(peers && peers.length > 0) }, [peers]);

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
      const unique = files.filter((f) => !existing.has(`${f.name}-${f.size}`)); // ignore duplicates
      return [...prev, ...unique];
    });
  }

  const removeFile = (name) =>
    setSelectedFiles((prev) => prev.filter((f) => f.name != name));

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
            <FileEntry key={index} name={file.name} onClick={() => removeFile(file.name)} />
          ))}
        </div>
        <button
          className="send-button" disabled={!canSend}
          onClick={startSending}>Send</button>
      </div>
    </div>
  );
}

function ErrorMessage({ message, onRetry }) {
  return (
    <div className="error-message">
      <p>{message}</p>
        {onRetry !== undefined &&
          <button className="retry-button">
            <RetryIcon className="retry-icon" />
          </button>
        }
    </div>
  );
}

export default function TransferPane() {
  const peers = useContext(PeersContext);
  const [selectedPeers, setSelectedPeers] = useState([]);
  const [selectedFiles, setSelectedFiles] = useState([]);
  const [sending, setSending] = useState(true);

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

  const startSending = async () => {
    setSending(true);
    await sendFiles();
  };

  const stopSending = () => {
    setSending(false);
  };

  return (
    <div className="inner-content">
      {!sending &&
        <FileAndPeerSelection
          peers={peers} startSending={startSending} setSelectedFiles={setSelectedFiles}
          selectedFiles={selectedFiles} setSelectedPeers={setSelectedPeers}
        />
      }
      {sending &&
        <div class="inner-content">
          <div class="status-top-row">
            <button onClick={() => stopSending()}> Cancel </button>
            <h1> Sending </h1>
          </div>
          <div className="progress-container">
            <FileEntry name={"File A"} progress={0.5} />
            <FileEntry name={"File B"} progress={0.4} />
            <FileEntry name={"File C"} progress={0.8} />
          </div>
          <div className="error-tray">
            <ErrorMessage message={"this is the first error!"}/>
            <ErrorMessage
              message={"this is the second error!"} onRetry={() => console.log("hello!")}/>
          </div>
        </div>
      }
    </div>
  );
}
