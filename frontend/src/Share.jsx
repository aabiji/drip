import { useEffect, useState } from "react";
import uploadIcon from "./assets/upload.svg";

import { EventsOn } from "../wailsjs/runtime/runtime";
import { GetPeers } from "../wailsjs/go/main/App";

export default function SharePane() {
  const [peers, setPeers] = useState([]);
  const [canSend, setCanSend] = useState(true);

  // Fetch list of peers from the backend
  useEffect(() => { setCanSend(peers && peers.length > 0) }, [peers]);
  useEffect(() => {
    EventsOn("peers-updated", () => GetPeers().then((names) => setPeers(names)));
  }, []);

  return (
    <div className="inner-content">
      <div className="upper-container">
        <h3> Send to </h3>
        {canSend ? (
          <div className="devices-container">
            {peers.map((name, index) => (
              <div className="device-entry" key={index}>
                <label className="custom-checkbox">
                  <input type="checkbox" className="checkbox" />
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
        <div className="file-input-container">
          <label className={canSend ? "file-label" : "file-label disabled"}>
            <img src={uploadIcon} className="upload-icon" alt="Upload" />
            <p>Drag and drop or choose files</p>
            <input type="file" disabled={!canSend} />
          </label>
        </div>
        <button className="send-button" disabled={!canSend}>
          Send
        </button>
      </div>
    </div>
  );
}
