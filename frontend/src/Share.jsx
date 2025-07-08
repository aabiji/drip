import uploadIcon from "./assets/upload.svg";

export default function SharePane() {
  const deviceNames = [
    "Device A", "Device B", "Device C", "Device D",
  ];

  return (
    <div class="inner-content">
      <div class="upper-container">
        <h3> Send to </h3>
        <div class="devices-container">
          {deviceNames.map((name, index) => (
            <div class="device-entry" key={index}>
              <label class="custom-checkbox">
                <input type="checkbox" class="checkbox" />
                <span class="fake-checkbox"></span>
              </label>
              <p>{name}</p>
            </div>
          ))}
        </div>
      </div>


      <div class="upload-container">
        <div class="file-input-container">
          <label class="file-label">
            <img src={uploadIcon} class="upload-icon" alt="Upload" />
            <p>Drag and drop or choose files</p>
            <input type="file" />
          </label>
        </div>
        <button class="send-button">Send</button>
      </div>
    </div>
  );
}
