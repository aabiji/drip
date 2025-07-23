import { createContext, useEffect, useState } from "react";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { GetPeers, GetSettings, SaveSettings } from "../wailsjs/go/main/App";

export const PeersContext = createContext([]);
export const ErrorContext = createContext([]);
export const SettingsContext = createContext({});
export const TransferContext = createContext(null);

export default function State({ children }) {
  // periodically fetch list of peers from the backend
  const [peers, setPeers] = useState([]);
  useEffect(() => {
    EventsOn("PEERS_UPDATED", () =>
      GetPeers().then((names) => setPeers(names)),
    );
  }, []);

  // transfer info
  const [selectedPeers, setSelectedPeers] = useState([]);
  const [selectedFiles, setSelectedFiles] = useState([]);
  const [transferIds, setTransferIds] = useState([]);
  const [sending, setSending] = useState(false);

  // error system
  const [errors, setErrors] = useState([]);

  const addError = (message) =>
    setErrors((prev) => [...prev, { id: crypto.randomUUID(), message }]);

  const removeError = (id) =>
    setErrors((prev) => prev.filter((error) => error.id !== id));

  // settings
  let loaded = false;
  const [theme, setTheme] = useState("light");
  const [trustPeers, setTrustPeers] = useState(true);
  const [showNotifications, setShowNotifications] = useState(true);
  const [downloadFolder, setDownloadFolder] = useState("");

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
  }, [theme]);

  useEffect(() => {
    if (!loaded) return;
    SaveSettings({ theme, trustPeers, showNotifications, downloadFolder });
  }, [theme, trustPeers, showNotifications, downloadFolder]);

  useEffect(() => {
    (async () => {
      const settings = await GetSettings();
      setTheme(settings.theme);
      setTrustPeers(settings.trustPeers);
      setShowNotifications(settings.showNotifications);
      setDownloadFolder(settings.downloadFolder);
      loaded = true;
    })();
  }, []);

  return (
    <SettingsContext.Provider
      value={{
        theme,
        setTheme,
        trustPeers,
        setTrustPeers,
        showNotifications,
        setShowNotifications,
        downloadFolder,
        setDownloadFolder,
      }}
    >
      <ErrorContext.Provider value={{ errors, addError, removeError }}>
        <PeersContext.Provider value={peers}>
          <TransferContext.Provider
            value={{
              sending,
              setSending,
              transferIds,
              setTransferIds,
              selectedPeers,
              setSelectedPeers,
              selectedFiles,
              setSelectedFiles,
            }}
          >
            {children}
          </TransferContext.Provider>
        </PeersContext.Provider>
      </ErrorContext.Provider>
    </SettingsContext.Provider>
  );
}
