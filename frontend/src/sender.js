import { StartFileTransfer, SendFileChunk, SendCancelSignal} from "../wailsjs/go/main/App";

export class Transfer {
  constructor(file, id, recipient) {
    this.id = id;
    this.recipient = recipient;
    this.file = file;

    this.done = false;
    this.cancelled = false;
    this.hadError = false;

    this.amountSent = 0;
    this.sentValue = undefined;

    this.numRetries = 0;
    this.maxRetries = 10;
    this.lastResponseTime = Date.now();
  }

  async readChunk() {
    // Send 256 kb chunks so that we stay below webrtc's message limit
    const chunkSize = 256 * 1024;
    if (this.amountSent >= this.file.size) {
      this.done = true;
      return;
    }

    const end = Math.min(this.amountSent + chunkSize, this.file.size);
    const slice = this.file.slice(this.amountSent, end);

    const chunk = await new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(Array.from(new Uint8Array(reader.result)));
      reader.onerror = () => {
        this.hadError = true;
        reject(new Error(`Couldn't read ${this.id}`));
      }
      reader.readAsArrayBuffer(slice);
    });
    this.sentValue = chunk;
  }

  // See downloader.go for the object fields, and app.go for the exported functions
  async sendInfo() {
    try {
      await StartFileTransfer({
        transferId: this.id, recipient: this.recipient,
        name: this.file.name, size: this.file.size
      });
    } catch (error) {
      this.hadError = true;
      throw new Error(`Failed to start transferring ${this.id}`);
    }
  }

  async sendChunk() {
    try {
      await SendFileChunk({
        transferId: this.id, recipient: this.recipient,
        offset: this.amountSent, data: this.sentValue,
      });
    } catch (error) {
      this.hadError = true;
      throw new Error(`Failed to send chunk for ${this.id}`);
    }
  }

  async sendCancel() {
    try {
      await SendCancelSignal({ transferId: this.id, recipient: this.recipient });
      this.cancelled = true;
    } catch (error) {
      this.hadError = true;
      throw new Error(`Failed to cancel transferring ${this.id}`);
    }
  }
}

export let TRANSFERS = {}; // TODO: load and save this

export async function startTransfer(selectedFiles, selectedPeers, setTransferIds) {
  TRANSFERS = {}; // clear all previous transfers

  for (const file of selectedFiles) {
    const parts = file.name.split('.');
    const [base, extension] = [parts[0], parts[parts.length - 1]];

    for (const peer of selectedPeers) {
      const transferId = `${base}-${Math.floor(Math.random() * 100)}.${extension}`;
      const clone = new File([file], transferId, { type: file.type });
      const transfer = new Transfer(clone, transferId, peer);

      TRANSFERS[transferId] = transfer;
      setTransferIds(prev => [...prev, transferId]);
      await transfer.sendInfo();
    }
  }
}

export async function cancelTransfers() {
  for (const id in TRANSFERS) {
    if (TRANSFERS[id].hadError || TRANSFERS[id].cancelled)
      continue;
    await TRANSFERS[id].sendCancel();
  }
}

export async function handleResponse(response, setTransferIds) {
  const json = JSON.parse(atob(response["data"]));
  const transferId = json["transferId"];

  if (json["cancelled"]) {
    delete TRANSFERS[transferId];
    return;
  }

  TRANSFERS[transferId].lastResponseTime = Date.now();
  TRANSFERS[transferId].amountSent += json["amountReceived"];

  await sendMessage(transferId, true);
  setTransferIds(prev => [...prev]); // force rerender
}

export async function sendMessage(transferId, advance) {
  const transfer = TRANSFERS[transferId];

  if (transfer.cancelled) {
    await transfer.sendCancel();
    return;
  }

  if (advance == false && transfer.sentValue === undefined) {
    await transfer.sendInfo();
    return;
  }

  if (advance) {
    await transfer.readChunk();
    if (transfer.done) return;
  }

  await transfer.sendChunk();
}

export async function resendMessages(setTransferIds) {
  for (const id in TRANSFERS) {
    const transfer = TRANSFERS[id];
    if (transfer.done || transfer.hadError) continue;

    const elapsedSeconds = (Date.now() - transfer.lastResponseTime) / 1000;
    const needToRetry = elapsedSeconds >= 10;
    if (!needToRetry) continue;

    transfer.numRetries += 1;
    if (transfer.numRetries >= transfer.maxRetries) {
      delete TRANSFERS[id];
      setTransferIds(prev => prev.filter(transferId => transferId != id));
      throw new Error(`Max retries for ${id} exceeded, cancelling transfer.`);
    } else {
      await sendMessage(id, false);
    }
  }
}