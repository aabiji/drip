import { useContext } from "react";
import { SettingsContext } from "./State";
import { SelectDowloadFolder } from "../wailsjs/go/main/App";
import { Moon, Sun } from "feather-icons-react";

export default function SettingsView() {
  const {
      theme, setTheme,
      trustPeers, setTrustPeers,
      showNotifications, setShowNotifications,
      downloadFolder, setDownloadFolder
  } = useContext(SettingsContext);

  const pickDownloadPath = async () => setDownloadFolder(await SelectDowloadFolder());

  const startYear = 2025;
  const currentYear = new Date().getFullYear();
  const copyright =
    startYear == currentYear ? `${startYear}` : `${startYear}-${currentYear}`;

  return (
    <div className="content">
      <div className="row">
        <p className="input-label">Toggle theme</p>
        <button
          className="icon-button"
          onClick={() => setTheme(theme == "light" ? "dark" : "light")}
        >
          {theme == "light"
            ? <Moon className="icon-button-svg" />
            : <Sun className="icon-button-svg" />
          }
        </button>
      </div>

      <div className="row">
        <p>Trust peers</p>
        <label className="custom-checkbox">
          <input
            type="checkbox" className="checkbox"
            onChange={() => setTrustPeers(!trustPeers)} />
          <span className="fake-checkbox"></span>
        </label>
      </div>

      <div className="row">
        <p>Show notifications</p>
        <label className="custom-checkbox">
          <input
            type="checkbox" className="checkbox"
            onChange={() => setShowNotifications(!showNotifications)} />
          <span className="fake-checkbox"></span>
        </label>
      </div>

      <div className="row">
        <p>Download folder</p>
        <button class="folder-path-input"
          onClick={() => pickDownloadPath()}>{downloadFolder}</button>
      </div>

      <p className="copyright">Made with ❤️ by Abigail Adegbiji {copyright}</p>
    </div>
  );
}
