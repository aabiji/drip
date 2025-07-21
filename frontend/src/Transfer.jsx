import { useContext, useEffect, useState } from "react";

import { EventsOn } from "../wailsjs/runtime/runtime";

 // see downloader.go for the json schemas to these function arguments
import { RequestSessionAuth, CancelSession, SendFileChunk } from "../wailsjs/go/main/App";

import { ErrorContext, TransferContext } from "./State";
import TransferSelection from "./TransferSelection";

// Edge cases:
// - Peer disconnects during transfer
// - We disconnect during transfer
// - Can't read from file

function randomizeFilename(filename) {
  const parts = filename.split('.');
  const [base, extension] = [parts[0], parts[parts.length - 1]];
  const random = crypto.randomUUID();
  return `${base}-${random}.${extension}`;
}

class Transfer {
  constructor(file, id, recipient) {
    this.id = id;
    this.recipient = recipient;
    this.file = file;
    this.amountSent = 0;
    this.done = false;
  }

  async start() {
    const reader = this.file.stream().getReader();

    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        this.done = true;
        console.log(`Done sending ${this.id}`);
        break;
      }

      console.log(`Sending chunk for ${this.id}`);
      await SendFileChunk({
        transferId: this.id,
        offset: this.amountSent,
        recipient: this.recipient,
        data: Array.from(value),
      });
      this.amountSent += value.length;
    }
  }
}

class Session {
  constructor(files, recipients) {
    this.id = crypto.randomUUID();

    // map each peer to whether they have authorized or not
    this.recipients = recipients.reduce((map, key) => {
      map[key] = false;
      return map;
    }, {});

    this.transfers = {};
    for (const file of files) {
      for (const peer of recipients) {
        const transferId = randomizeFilename(file.name);
        const copy = new File([file], transferId, { type: file.type });
        this.transfers[transferId] = new Transfer(copy, transferId, peer);
      }
    }
  }

  // return true if all the recipients have authorized the batch of transfers
  fullyAuthorized() { return Object.values(this.recipients).every(Boolean); }

  async requestAuthorization() {
    let payload = {
      sessionId: this.id,
      recipients: Object.keys(this.recipients),
      transfers: Object.values(this.transfers).map(t => ({
        transferId: t.id,
        recipient: t.recipient,
        size: t.file.size
      }))
    };
    await RequestSessionAuth(payload);
    console.log(`Requesting auth for ${this.id}`);
  }

  async cancel() {
    await CancelSession({ sessionId: this.id, recipients: Object.keys(this.recipients) });
    console.log(`Cancelling ${this.id}`);
  }

  async handleResponse(response) {
    const json = JSON.parse(atob(response["data"]));

    if (!json.accepted) {
      await this.cancel();
      throw new Error(`${response.senderId} does not allow the transfer`);
    }

    this.recipients[response.senderId] = true;
    if (this.fullyAuthorized()) {
      console.log(`${this.id} is fully authorized!`);
      for (const id in this.transfers) {
        this.transfers[id].start();
      }
    }
  }
}

// Only allowing the user to transfer one batch at a time
let CURRENT_SESSION = undefined;

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
      if (sending) {
        CURRENT_SESSION = new Session(selectedFiles, selectedPeers);
        await CURRENT_SESSION.requestAuthorization();
      }
    });
  }, [sending]);

  useEffect(() => {
    errorHandler(async () => {
      if (cancel) {
        await CURRENT_SESSION.cancel();
        setSending(false);
        setCancel(false);
      }
    });
  }, [cancel]);

  // TODO: resuming transfers???
  useEffect(() => {
    const stop = EventsOn("SESSSION_RESPONSE",
      (data) => errorHandler(async () => await CURRENT_SESSION.handleResponse(data)));
    return () => stop();
  }, []);

  return (
    <div className="content">
      {!sending &&
        <TransferSelection
          setSending={setSending}
          selectedPeers={selectedPeers} setSelectedPeers={setSelectedPeers}
          selectedFiles={selectedFiles} setSelectedFiles={setSelectedFiles} />}
      {sending &&
        <div className="content">
          {(() => {
            const done = false;//Object.values(CURRENT_SESSION.transfers).every(t => t.done);
            const failed = false; // TODO: handle error handling
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
              const t = CURRENT_SESSION.transfers[id];
              if (t === undefined) return null;
              return <FileEntry key={id} name={t.file.name}
                        recipient={t.recipient} progress={t.amountSent / t.file.size} />;
            })}
          </div>
        </div>
      }
    </div>
  );
}
