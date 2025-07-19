import { useContext, useEffect, useState, useRef } from "react";
import { EventsOn } from "../wailsjs/runtime/runtime";

import { PeersContext, TransferContext } from "./StateProvider";
import { TRANSFER_RESPONSE } from "./constants";
import * as sender from "./sender";

import { ReactComponent as UploadIcon } from "./assets/upload.svg";

function FileEntry({ name, progress, onClick, recipient }) {
  const barElement = useRef();
  const [full, setFull] = useState(false);
  const msg = recipient !== undefined ? `Sending ${name} to ${recipient}` : name;

  useEffect(() => {
    if (progress !== undefined) {
      barElement.current.style.width = `${Math.min(progress, 1.0) * 100}%`;
      setFull(progress >= 1.0);
    }
  }, [progress]);

  return (
    <div className={!full ? "file-entry" : "file-entry full"}>
      <div className="inner">
        <p>{msg}</p>
        {onClick !== undefined && <button onClick={onClick}>x</button>}
      </div>
      {progress !== undefined && <div className="progress-bar" ref={barElement}></div>}
    </div>
  );
}

// FIXME: if we select file before peer, the button's diabled
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
    <div className="inner-content">
      <div className="upper-container">
        <h3> Send to </h3>
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
          onClick={() => setSending(true)}>Send</button>
      </div>
    </div>
  );
}

export default function TransferPane() {
  const {
    sending, setSending,
    transferIds, setTransferIds,
    selectedPeers, setSelectedPeers,
    selectedFiles, setSelectedFiles
  } = useContext(TransferContext);

  const [done, setDone] = useState(false);
  const [cancel, setCancel] = useState(false);

  useEffect(() => {
    if (sending)
      sender.startTransfer(selectedFiles, selectedPeers, setTransferIds);
    setSelectedFiles([]);
    setSelectedPeers([]);
  }, [sending]);

  useEffect(() => {
    if (cancel) {
      sender.cancelTransfers();
      setSending(false);
      setDone(false);
      setCancel(false);
    }
  }, [cancel]);

  // TODO: resuming transfers???
  useEffect(() => {
    const cancelListener = EventsOn(TRANSFER_RESPONSE,
      (data) => sender.handleResponse(data, setTransferIds, setDone));

    const intervalId = setInterval(async () =>
      await sender.resendMessages(setTransferIds, setDone), 10000);
    return () => {
      cancelListener();
      clearInterval(intervalId);
    }
  }, []);

  return (
    <div className="inner-content">
      {!sending &&
        <FileAndPeerSelection
          setSending={setSending}
          selectedPeers={selectedPeers} setSelectedPeers={setSelectedPeers}
          selectedFiles={selectedFiles} setSelectedFiles={setSelectedFiles} />}
      {sending &&
        <div className="inner-content">
          <div className="status-top-row">
            <button onClick={() => done ? setSending(false) : setCancel(true)}>
              {done ? "Back" : "Cancel"}
            </button>
            <h1> {done ? "Done sending!" : "Sending..."} </h1>
          </div>

          <div className="progress-container">
            {transferIds.map(id => {
              const t = sender.TRANSFERS[id];
              if (t === undefined) return null;
              return <FileEntry key={id} name={t.file.name}
                        recipient={t.recipient} progress={t.amountSent / t.file.size} />;
            })}
          </div>
          <div className="error-tray">{/* TODO: error messages go here */}</div>
        </div>
      }
    </div>
  );
}
