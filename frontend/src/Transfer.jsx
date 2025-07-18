import { useContext, useEffect, useState, useRef } from "react";

import { EventsOn } from "../wailsjs/runtime/runtime";
import { StartFileTransfer, SendFileChunk } from "../wailsjs/go/main/App";

import { PeersContext } from "./StateProvider";
import { TransferContext } from "./StateProvider";

import { TRANSFER_STATE } from "./constants";

import { ReactComponent as UploadIcon } from "./assets/upload.svg";

function FileEntry({ name, progress, onClick, recipient }) {
  const barElement = useRef();
  const [full, setFull] = useState(false);
  const msg = recipient !== undefined ? `Sending ${name} to ${recipient}` : name;

  useEffect(() => {
    if (progress !== undefined) {
      const total = barElement.current.parentElement.offsetWidth - 2;
      barElement.current.style.width = `${progress * total}px`;
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
  }, [peers, selectedFiles]);

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
        {havePeers ? (
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

class Transfer {
  constructor(file, id, recipient) {
    this.id = id;
    this.recipient = recipient;

    this.filename = file.name;
    this.filesize = file.size;
    this.fileReader = file.stream().getReader();

    this.amountSent = 0;
    this.sentValue = undefined;
    this.lastResponseTime = Date.now();
  }

  async sendInfo() { // see downloader.go for object params
    const info = {
      transferId: this.id, recipient: this.recipient,
      name: this.filename, size: this.filesize
    };
    await StartFileTransfer(info);
  }

  async sendChunk() { // see downloader.go for object params
    const info = {
      transferId: this.id, recipient: this.recipient,
      data: this.sentValue, offset: this.amountSent
    }
    await SendFileChunk(info);
  }
}

// TODO: file transfering with large files falls apart
let TRANSFERS = {}; // TODO: load and save this

export default function TransferPane() {
  const {
    sending, setSending,
    transferIds, setTransferIds,
    selectedPeers, setSelectedPeers,
    selectedFiles, setSelectedFiles
  } = useContext(TransferContext);

  const startTransfer = async () => {
    for (const file of selectedFiles) {
      for (const peer of selectedPeers) {
        const transferId = `${file.name}-${Math.floor(Math.random() * 100)}`;
        const transfer = new Transfer(file, transferId, peer);
        TRANSFERS[transferId] = transfer;
        setTransferIds(prev => [...prev, transferId]);
        await transfer.sendInfo();
      }
    }
  }

  const sendChunk = async (transferId, advance) => {
    const transfer = TRANSFERS[transferId];

    // read the next file chunk
    if (advance) {
      const { value, done } = await transfer.fileReader.read();
      if (done) {
        transfer.done = true;
        return;
      }

      console.log("sending chunk");
      transfer.sentValue = Array.from(value);
    } else {
      console.log("resending chunk");
    }

    // resend the file transfer info message
    if (advance == false && transfer.sentValue === undefined) {
      console.log("resending transfer info");
      await transfer.sendInfo();
      return;
    }

    await transfer.sendChunk();
  }

  const handleTransferState = async (response) => {
    const json = JSON.parse(atob(response["data"]));
    const transferId = json["transferId"];

    TRANSFERS[transferId].lastResponseTime = Date.now();
    TRANSFERS[transferId].amountSent += json["amountReceived"];

    setTransferIds(prev => [...prev]); // force rerender
    await sendChunk(transferId, true);
  }

  // Resend a file chunk to peers who haven't responded in
  // a while, assuming that in that case, they didn't get the chunk
  const resendChunks = async () => {
    for (const id in TRANSFERS) {
      const transfer = TRANSFERS[id];
      if (transfer.done) continue;

      const [now, timeoutSeconds] = [Date.now(), 10];
      const elapsedSeconds = (now - transfer.lastResponseTime) / 1000;
      if (elapsedSeconds >= timeoutSeconds)
        await sendChunk(id, false);
    }
  }

  const sendFiles = async () => {
    if (selectedFiles.length == 0 || selectedPeers.length == 0) return;
    await startTransfer();
    setSelectedFiles([]);
    setSelectedPeers([]);
  };

  useEffect(() => {
    const startSending = async () => await sendFiles();
    if (sending) startSending();
    // TODO: else, stop the file transfers
  }, [sending]);

  // TODO: resuming transfers???
  useEffect(() => {
    const cancelListener =
      EventsOn(TRANSFER_STATE, (data) => handleTransferState(data));

    const intervalId = setInterval(async () => await resendChunks(), 10000);
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
            <button onClick={() => setSending(false)}> Cancel </button>
            <h1> Sending </h1>
          </div>
          <div className="progress-container">
            {transferIds.map(id => {
              const t = TRANSFERS[id];
              return <FileEntry key={id} name={t.filename}
                        recipient={t.recipient} progress={t.amountSent / t.filesize} />;
            })}
          </div>
          <div className="error-tray">{/* TODO: error messages go here */}</div>
        </div>
      }
    </div>
  );
}
