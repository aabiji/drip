import { useContext, useEffect, useState, useRef } from "react";
import { EventsOn } from "../wailsjs/runtime/runtime";

import { ErrorContext, PeersContext, TransferContext } from "./State";
import { TRANSFER_RESPONSE } from "./constants";
import * as sender from "./sender";

import { ReactComponent as UploadIcon } from "./assets/upload.svg";

function FileEntry({ name, progress, onClick, recipient, error }) {
  const barElement = useRef();
  const [full, setFull] = useState(false);
  const [msg, setMsg] = useState(
    recipient !== undefined ? `Sending ${name} to ${recipient}` : name
  );

  useEffect(() => {
    if (error && recipient !== undefined) {
      progress = undefined;
      barElement.current.style.width = "0px";
      setMsg(`Failed to send ${name} to ${recipient}`);
      return;
    }

    if (progress !== undefined) {
      barElement.current.style.width = `${Math.min(progress, 1.0) * 100}%`;
      setFull(progress >= 1.0);
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

function FileAndPeerSelection({
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

  const addNonDuplicateFiles = (files) => {
    setSelectedFiles((prev) => {
      const existing = new Set(prev.map((f) => `${f.name}-${f.size}`));
      const unique = files.filter((f) => !existing.has(`${f.name}-${f.size}`));
      return [...prev, ...unique];
    });
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
            <UploadIcon className="upload-icon" />
            <p>Drag and drop or choose files</p>
            <input type="file" onChange={(event) => addNonDuplicateFiles(Array.from(event.target.files))} />
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

// TODO: handle the edge case where the peer disconnects during transferring...
export default function TransferView() {
  const {
    sending, setSending,
    transferIds, setTransferIds,
    selectedPeers, setSelectedPeers,
    selectedFiles, setSelectedFiles
  } = useContext(TransferContext);

  const [cancel, setCancel] = useState(false);

  const { addError } = useContext(ErrorContext);
  const errorHandler = async (func) => {
    try {await func(); } catch (error) { addError(error.toString()); }
  }

  useEffect(() => {
    errorHandler(async () => {
      if (sending)
        sender.startTransfer(selectedFiles, selectedPeers, setTransferIds);
      setSelectedFiles([]);
      setSelectedPeers([]);
    });
  }, [sending]);

  useEffect(() => {
    errorHandler(async () => {
      if (cancel) {
        await sender.cancelTransfers();
        setSending(false);
        setCancel(false);
      }
    });
  }, [cancel]);

  // TODO: resuming transfers???
  useEffect(() => {
    const cancelListener = EventsOn(TRANSFER_RESPONSE,
      (data) => errorHandler(async () => await sender.handleResponse(data, setTransferIds)));

    const intervalId = setInterval(() =>
      errorHandler(async () => await sender.resendMessages(setTransferIds)),
    10000);
    return () => {
      cancelListener();
      clearInterval(intervalId);
    }
  }, []);

  return (
    <div className="content">
      {!sending &&
        <FileAndPeerSelection
          setSending={setSending}
          selectedPeers={selectedPeers} setSelectedPeers={setSelectedPeers}
          selectedFiles={selectedFiles} setSelectedFiles={setSelectedFiles} />}
      {sending &&
        <div className="content">
          {(() => {
            const done = Object.values(sender.TRANSFERS).every(transfer => transfer.done);
            const failed = Object.values(sender.TRANSFERS).every(transfer => transfer.hadError);
            return (
              <div className="status-top-row">
                <button onClick={() => done ? setSending(false) : setCancel(true)}>
                  {done || failed ? "Back" : "Cancel"}
                </button>
                <h1> {done ? "Done sending!" : failed ? "Sending failed" : "Sending"} </h1>
              </div>
            );
          })()}

          <div className="progress-container">
            {transferIds.map(id => {
              const t = sender.TRANSFERS[id];
              if (t === undefined) return null;
              return <FileEntry key={id} name={t.file.name} error={t.hadError}
                        recipient={t.recipient} progress={t.amountSent / t.file.size} />;
            })}
          </div>
        </div>
      }
    </div>
  );
}
