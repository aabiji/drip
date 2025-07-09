import { useState } from "react";

import sunIcon from "./assets/sun.svg";
import moonIcon from "./assets/moon.svg";

export default function Settings() {
  const startYear = 2025;
  const currentYear = new Date().getFullYear();
  const copyright =
    startYear == currentYear ? `${startYear}` : `${startYear}-${currentYear}`;

  const [downloadPath, setDownloadPath] = useState("~/Downloads/");

  const [isLightMode, setIsLightMode] = useState(true);

  return (
    <div className="inner-content">
      <div className="row">
        <p className="input-label">Toggle theme</p>
        <button
          className="icon-button"
          onClick={() => setIsLightMode(!isLightMode)}
        >
          <img src={isLightMode ? moonIcon : sunIcon} alt="Toggle Theme" />
        </button>
      </div>

      <div className="row">
        <p className="input-label">Download folder</p>
        <label className="folder-label">
          <p className="path">{downloadPath}</p>
          <input type="file" webkitdirectory className="folder-path-input" />
        </label>
      </div>

      <p className="copyright">Made with ❤️ by Abigail Adegbiji {copyright}</p>
    </div>
  );
}
