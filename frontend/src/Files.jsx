import fileIcon from "./assets/file.svg";
import folderIcon from "./assets/folder.svg";

export default function ReceivedFiles() {
  const transfers = [
    { path: "/path/to/fileA",   sentFrom: "Device A", folder: false },
    { path: "/path/to/fileB",   sentFrom: "Device B", folder: false },
    { path: "/path/to/folderA", sentFrom: "Device C", folder: true },
  ];

  return (
    <div class="transfer-grid">

      {transfers.map((transfer, index) => (
        <div class="transfer-card" key={index}>
          <img src={transfer.folder ? folderIcon : fileIcon} class="banner" />
          <div class="info">
            <b> {transfer.path} </b>
            <p> Sent from {transfer.sentFrom} </p>
          </div>
        </div>
      ))}

    </div>
  );
}
