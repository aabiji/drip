import { ReactComponent as FileIcon } from "./assets/file.svg";
import { ReactComponent as FolderIcon } from "./assets/folder.svg";

export default function ReceivedFiles() {
  const transfers = [
    { path: "/path/to/fileA", sentFrom: "Device A", folder: false },
    { path: "/path/to/fileB", sentFrom: "Device B", folder: false },
    { path: "/path/to/folderA", sentFrom: "Device C", folder: true },
  ];

  return (
    <div className="transfer-grid">
      {transfers.map((transfer, index) => (
        <div className="transfer-card" key={index}>
          {transfer.folder
            ? <FolderIcon className="banner" />
            : <FileIcon className="banner" />
          }
          <div className="info">
            <b> {transfer.path} </b>
            <p> Sent from {transfer.sentFrom} </p>
          </div>
        </div>
      ))}
    </div>
  );
}
