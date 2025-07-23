import { useContext, useEffect, useState } from "react";
import { ArrowLeft } from "feather-icons-react";
import { EventsOn } from "../wailsjs/runtime/runtime";

import {
  // see downloader.go for the json schemas to these function arguments
  CancelSession,
  RequestSessionAuth,
  SendFileChunk,
} from "../wailsjs/go/main/App";

import { ErrorContext, TransferContext } from "./State";
import TransferSelection, { FileEntry } from "./Selection";

function randomizeFilename(filename) {
  const parts = filename.split(".");
  const [base, extension] = [parts[0], parts[parts.length - 1]];
  const random6DigitNum = Math.floor(100000 + Math.random() * 900000);
  return `${base}-${random6DigitNum}.${extension}`;
}

class Transfer {
  constructor(file, id, sessionId, recipient) {
    this.id = id;
    this.sessionId = sessionId;
    this.recipient = recipient;
    this.file = file;
    this.amountSent = 0;
    this.done = false;
    this.hadError = false;
  }

  async start(setTransferIds, addError) {
    const reader = this.file.stream().getReader();

    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        this.done = true;
        break;
      }

      try {
        await SendFileChunk({
          transferId: this.id,
          offset: this.amountSent,
          recipient: this.recipient,
          data: Array.from(value),
        });

        this.amountSent += value.length;
        setTransferIds((prev) => [...prev]); // force react to rerender
      } catch (error) {
        this.hadError = true;
        addError(error.toString());
        break;
      }
    }
  }
}

class Session {
  constructor(files, recipients, setTransferIds) {
    this.id = crypto.randomUUID();
    this.hadError = false;

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
        this.transfers[transferId] = new Transfer(
          copy,
          transferId,
          this.id,
          peer,
        );
      }
    }
    setTransferIds(Object.keys(this.transfers));
  }

  // return true if all the recipients have authorized the batch of transfers
  fullyAuthorized() {
    return Object.values(this.recipients).every(Boolean);
  }

  async requestAuthorization() {
    try {
      await RequestSessionAuth({
        sessionId: this.id,
        recipients: Object.keys(this.recipients),
        transfers: Object.values(this.transfers).map((t) => ({
          sessionId: this.id,
          transferId: t.id,
          recipient: t.recipient,
          size: t.file.size,
        })),
      });
    } catch (error) {
      this.hadError = true;
      addError(error.toString());
    }
  }

  async cancel(addError) {
    try {
      await CancelSession({
        sessionId: this.id,
        recipients: Object.keys(this.recipients),
      });
    } catch (error) {
      this.hadError = true;
      addError(error.toString());
    }
  }

  async handleResponse(response, setTransferIds, addError) {
    const json = JSON.parse(atob(response["data"]));

    if (!json.accepted) {
      await this.cancel(addError);
      this.hadError = true;
      addError(`${response.senderId} does not allow the transfer`);
      return;
    }

    this.recipients[response.senderId] = true;
    if (this.fullyAuthorized()) {
      for (const id in this.transfers) {
        this.transfers[id].start(setTransferIds, addError);
      }
    }
  }
}

// Only allowing the user to transfer one batch at a time
let CURRENT_SESSION = undefined;

export default function TransferView() {
  const {
    sending,
    setSending,
    transferIds,
    setTransferIds,
    selectedPeers,
    setSelectedPeers,
    selectedFiles,
    setSelectedFiles,
  } = useContext(TransferContext);

  const { addError } = useContext(ErrorContext);
  const [cancel, setCancel] = useState(false);

  useEffect(() => {
    if (sending) {
      CURRENT_SESSION = new Session(
        selectedFiles,
        selectedPeers,
        setTransferIds,
      );
      console.log("set", CURRENT_SESSION);
      CURRENT_SESSION.requestAuthorization(addError);
    } else {
      CURRENT_SESSION = undefined;
    }
  }, [sending]);

  useEffect(() => {
    if (cancel) {
      CURRENT_SESSION.cancel(addError);
      CURRENT_SESSION = undefined;
      setSending(false);
      setCancel(false);
    }
  }, [cancel]);

  useEffect(() => {
    const stop = EventsOn("SESSSION_RESPONSE", (data) => {
      if (CURRENT_SESSION)
        CURRENT_SESSION.handleResponse(data, setTransferIds, addError);
    });
    return () => stop();
  }, []);

  return (
    <div className="content">
      {!sending && (
        <TransferSelection
          setSending={setSending}
          selectedPeers={selectedPeers}
          setSelectedPeers={setSelectedPeers}
          selectedFiles={selectedFiles}
          setSelectedFiles={setSelectedFiles}
        />
      )}

      {sending && (
        <div className="content">
          {(() => {
            if (CURRENT_SESSION === undefined) return null;
            const authorized = CURRENT_SESSION.fullyAuthorized();
            const done = Object.values(CURRENT_SESSION.transfers).every(
              (t) => t.done,
            );
            const failed =
              CURRENT_SESSION.hadError ||
              Object.values(CURRENT_SESSION.transfers).some((t) => t.hadError);

            let msg = "Sending";
            if (failed) msg = "Sending failed";
            if (!authorized) msg = "Waiting for authorization";
            if (done) msg = "Done sending";

            return (
              <div className="status-top-row">
                <button
                  onClick={() => (done ? setSending(false) : setCancel(true))}
                >
                  <ArrowLeft class="settings-icon" />
                </button>
                <h1>{msg}</h1>
              </div>
            );
          })()}

          <div className="progress-container">
            {transferIds.map((id) => {
              if (CURRENT_SESSION === undefined) return null;
              const t = CURRENT_SESSION.transfers[id];
              if (t === undefined) return null;
              return (
                <FileEntry
                  key={id}
                  name={t.file.name}
                  error={t.hadError}
                  recipient={t.recipient}
                  progress={t.amountSent / t.file.size}
                />
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
