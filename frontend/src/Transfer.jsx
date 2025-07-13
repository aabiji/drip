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

/*
Ok, send file transfer start to backend:
- there's this FileTransfers service that holds all the peers
- it holds a cache of all the transfer info,
  creates a new transferId for each file transfer
  backend should send that transferId in its acknoledgement response
  ** no, the frontend creates the transferId and stores it in the header

type FileTransferInfo struct {
  name           string
  size           int64
  numChunks      int
  chunksReceived [numChunks]bool // chunks can come out of order

  tempFilePath   string
}

Map transferId to FileTransferInfo? Or just store it in a list.
Both the send and receiver should save this metadata to disk.
That way, any transfer goes as long as
we have a corresponding FileTransferInfo since we'd know how to handle it
Read from disk every time we open the app. Should share the info with the frontend

So when the backend gets a chunk, it should have a transferId to go along
with it, so the backend can update chunksReceived, and write the chunk.
Then send an acknoledgement response. On the sender's side, if we don't get
an acknoledgement in N seconds we know they probably didn't get it, and we
resend the chunk (like tcp's retransmission logic).

To write the output file on the recipient's device, we just create a new
file of a certain size (fallocate syscall), mmap and write the chunk.
The file gets closed if we've received all the chunks or the peer disconnects
or there's some other error. Each file should also get protected by a mutex,
to avoid race conditions.

We could checksum each hash to ensure its integrity (also pass along a checksum with the chunk,
and our checksum of the received chunk doesn't match its intended checksum we reject it
and pretend it was never sent).

```!
Each peer connection struct holds

fileTransferProgres map[string][]bool // maps the transfer id to a list of booleans

in the list of boolean, each index corresponds to whether we've received that chunk or not

and so the receiving peer sends that after receiving a chunk
```

Edge cases:
- what if a peer joins late?
*/

export default function TransferPane({ state }) {
  const streamFile = async (file) => {
    const chunkSize = 5 * 1024 * 1024; // 5 megabytes
    const numChunks = Math.ceil(file.size / chunkSize);
    const info = { recipients: state.selectedPeers, name: file.name, size: file.size, numChunks };

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
    if (state.selectedFiles.length == 0 || state.selectedPeers.length == 0) return;
    try {
      state.setPercentages(Array(state.selectedFiles.length).fill(0));
      for (const file of state.selectedFiles) {
        await streamFile(file);
      }
    } catch (error) {
      console.log("tell the user the error!");
    }
  };

  const startSending = async () => {
    state.setSending(true);
    await sendFiles();
  };

  const stopSending = () => {
    state.setPercentages([]);
    state.setSending(false);
    // TODO: stop file transfers
  };

  return (
    <div className="inner-content">
      {!state.sending && <FileAndPeerSelection state={state} />}
      {state.sending &&
        <div class="inner-content">
          <div class="status-top-row">
            <button onClick={() => stopSending()}> Cancel </button>
            <h1> Sending </h1>
          </div>
          <div className="progress-container">
            {state.percentages.map((p, index) =>
              <FileEntry name={state.selectedFiles[index].name} progress={p} />
            )}
          </div>
          <div className="error-tray">{/* TODO: error messages go here */}</div>
        </div>
      }
    </div>
  );
}
