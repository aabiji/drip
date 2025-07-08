import { useState } from "react";

import sunIcon from "./assets/sun.svg";
import moonIcon from "./assets/moon.svg";

export default function Settings() {
  const startYear = 2025;
  const currentYear = new Date().getFullYear();
  const copyright = startYear == currentYear ? `${startYear}`: `${startYear}-${currentYear}`;

  const [downloadPath, setDownloadPath] = useState("~/Downloads/");

  const [isLightMode, setIsLightMode] = useState(true);

  return (
    <div class="inner-content">
      <div class="row">
        <p class="input-label">Toggle theme</p>
        <button class="icon-button" onClick={() => setIsLightMode(!isLightMode)}>
          <img src={isLightMode ? moonIcon : sunIcon} alt="Toggle Theme" />
        </button>
      </div>

      <div class="row">
        <p class="input-label">Download folder</p>
        <label class="folder-label">
          <p class="path">{downloadPath}</p>
          <input type="file" webkitdirectory class="folder-path-input" />
        </label>
      </div>

      <p class="copyright">Made with ❤️ by Abigail Adegbiji {copyright}</p>
    </div>
  );
}

