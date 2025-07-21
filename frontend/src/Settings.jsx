import { useContext, useState } from "react";
import { ThemeContext } from "./State";

import { ReactComponent as SunIcon } from "./assets/sun.svg";
import { ReactComponent as MoonIcon } from "./assets/moon.svg";

export default function SettingsView() {
  const startYear = 2025;
  const currentYear = new Date().getFullYear();
  const copyright =
    startYear == currentYear ? `${startYear}` : `${startYear}-${currentYear}`;

  const [downloadPath, _setDownloadPath] = useState("~/Downloads/");

  const { theme, setTheme } = useContext(ThemeContext);

  return (
    <div className="content">
      <div className="row">
        <p className="input-label">Toggle theme</p>
        <button
          className="icon-button"
          onClick={() => setTheme(theme == "light" ? "dark" : "light")}
        >
          {theme == "light"
            ? <MoonIcon className="icon-button-svg" />
            : <SunIcon className="icon-button-svg" />
          }
        </button>
      </div>

      <div className="row">
        <p>Trust peers</p>
        <label className="custom-checkbox">
          <input
            type="checkbox" className="checkbox"
            onChange={(event) => selectPeer(event, name)} />
          <span className="fake-checkbox"></span>
        </label>
      </div>

      <div className="row">
        <p>Show popups</p>
        <label className="custom-checkbox">
          <input
            type="checkbox" className="checkbox"
            onChange={(event) => selectPeer(event, name)} />
          <span className="fake-checkbox"></span>
        </label>
      </div>

      <div className="row">
        <p className="input-label">Download folder</p>
        <label className="folder-label">
          <p className="path">{downloadPath}</p>
          <input type="file" webkitdirectory="true" className="folder-path-input" />
        </label>
      </div>

      <p className="copyright">Made with ❤️ by Abigail Adegbiji {copyright}</p>
    </div>
  );
}
