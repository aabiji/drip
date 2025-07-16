import { useContext, useEffect, useState, useRef } from "react";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { StartFileTransfer, SendFileChunk } from "../wailsjs/go/main/App";
import { PeersContext } from "./StateProvider";
import { TRANSFER_STATE } from "./constants";

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
    <div className={!full ? "file-entry" : "file-entry full"}>
      <div className="inner">
        <p>{name}</p>
        {onClick !== undefined && <button onClick={onClick}>x</button>}
      </div>
      {progress !== undefined && <div className="progress-bar" ref={barElement}></div>}
    </div>
  );
}

function FileAndPeerSelection({ state }) {
  const peers = useContext(PeersContext);
  const [canSend, setCanSend] = useState(true);

  // Fetch list of peers from the backend
  useEffect(() => { setCanSend(peers && peers.length > 0) }, [peers]);

  const selectPeer = (event, name) => {
    state.setSelectedPeers((prev) => {
      const list = event.target.checked
        ? [...prev, name]
        : prev.filter((peer) => peer != name);
      return list;
    });
  };

  const addNonDuplicateFiles = (files) => {
    state.setSelectedFiles((prev) => {
      const existing = new Set(prev.map((f) => `${f.name}-${f.size}`));
      const unique = files.filter((f) => !existing.has(`${f.name}-${f.size}`));
      return [...prev, ...unique];
    });
  };

  const removeFile = (name) =>
    state.setSelectedFiles((prev) => prev.filter((f) => f.name != name));

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
        <div
          className="file-input-container"
          onDrop={(event) => dropHandler(event)}
          onDragOver={(event) => dragOverHandler(event)}>
          <label className={canSend ? "file-label" : "file-label disabled"}>
            <UploadIcon className="upload-icon" />
            <p>Drag and drop or choose files</p>
            <input
              type="file" disabled={!canSend}
              onChange={(event) => addNonDuplicateFiles(Array.from(event.target.files))} />
          </label>
        </div>
        <div className="file-selection-container">
          {state.selectedFiles.map((file, index) => (
            <FileEntry key={index} name={file.name} onClick={() => removeFile(file.name)} />
          ))}
        </div>
        <button
          className="send-button" disabled={!canSend}
          onClick={() => state.setSending(true)}>Send</button>
      </div>
    </div>
  );
}

/*
error tray:
- we have a ErrorContext, where we write error messages
- then in app.jsx we loop through all the errors and
  render this component with absolute positioning
  ErrorTray
*/
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

class Transfer {
  constructor(file, id, recipient) {
    this.id = id;
    this.recipients = recipient;
    this.filename = file.name;
    this.filesize = file.size;

    this.amountSent = 0;
    this.currentChunk = undefined;
    this.lastResponseTime = Date.now();
    this.fileReader = file.stream().getReader();
  }
}

async function sendChunk(state, recipient, transferId, advance) {
  const transfer = state.transfers[recipient].find(t => t.id == transferId);

  if (advance) {
    const { value, done } = await transfer.fileReader.read();
    transfer.currentChunk = value;

    if (done) { // can remove the transfer from cache
      state.setTransfers(prev => prev.filter(t => t.id != transfer.id));
      return;
    }

    console.log("Sending chunk...", recipient, transferId)
  } else {
    console.log("Resending chunk...", recipient, transferId);
  }

  const chunk = { transferId, data: transfer.currentChunk }; // see downloader.go
  const ok = await SendFileChunk(chunk);
  if (!ok) {
    console.log("couldn't send chunk");
  }
}

async function startTransfer(state, file, recipients) {
  // see downloader.go
  const random = Math.floor(Math.random() * 100);
  const transferId = `${file.name}-${random}`;
  const info = { transferId, recipients, name: file.name, size: file.size };

  // keep record of the transfer
  for (const peerId of recipients) {
    const transfer = new Transfer(file, transferId, peerId);

    if (state.transfers[peerId] === undefined)
      state.setTransfers(prev => ({ ...prev, [peerId]: [transfer] }));
    else
      state.setTransfers(prev => ({ ...prev, [peerId]: [...prev[peerId], transfer] }));
  }

  const ok = await StartFileTransfer(info);
  if (!ok) {
    console.log("couldn't start file transfer");
    return;
  }

  console.log("Sending transfer info...", transferId);
}

// Resend a file chunk to peers who haven't responded in
// a while, assuming that in that case, they didn't get the chunk
async function resendChunks(state) {
  for (const peer in state.transfers) {
    for (const transfer of state.transfers[peer]) {
      const [now, timeoutSeconds] = [Date.now(), 3];
      const elapsedSeconds = (now - transfer.lastResponseTime) / 1000;
      if (elapsedSeconds >= timeoutSeconds)
        await sendChunk(state, peer, transfer.id, false);
    }
  }
}

// handle a transfer state response we get from a peer
async function handleTransferState(state, response) {
  const peerId = response["SenderId"];
  const json = JSON.parse(atob(response["Data"]));

  const transfer = state.transfers[peerId].find(t => t.id == json["transferId"]);
  transfer.amountSent += json["amountReceived"];
  transfer.lastResponseTime = Date.now();

  console.log("Handling peer response...", transfer);

  await sendChunk(state, peerId, json["transferId"], true);
}

export default function TransferPane({ state }) {
  const sendFiles = async () => {
    if (state.selectedFiles.length == 0 || state.selectedPeers.length == 0) return;

    try {
      state.setPercentages(Array(state.selectedFiles.length).fill(0));
      for (const file of state.selectedFiles) {
        await startTransfer(file, state.selectedPeers);
      }
    } catch (error) {
      console.log("tell the user the error!");
    }
  };

  // TODO: resuming transfers???

  setInterval(async () => await resendChunks(state), 10000);

  useEffect(() => {
    EventsOn(TRANSFER_STATE, (data) => handleTransferState(data));
  }, []);

  useEffect(() => {
    const startSending = async () => await sendFiles();

    if (state.sending)
      startSending();
    else {
      state.setPercentages([]);
      state.setSending(false);
      // TODO: stop file transfers
    }
  }, [state.sending]);

  return (
    <div className="inner-content">
      {!state.sending && <FileAndPeerSelection state={state} />}
      {state.sending &&
        <div className="inner-content">
          <div className="status-top-row">
            <button onClick={() => state.setSending(false)}> Cancel </button>
            <h1> Sending </h1>
          </div>
          <div className="progress-container">
            {state.percentages.map((p, index) =>
              <FileEntry key={index} name={state.selectedFiles[index].name} progress={p} />
            )}
          </div>
          <div className="error-tray">{/* TODO: error messages go here */}</div>
        </div>
      }
    </div>
  );
}
