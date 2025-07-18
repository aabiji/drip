import { createContext, useEffect, useState } from "react";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { GetPeers } from "../wailsjs/go/main/App";
import { PEERS_UPDATED } from "./constants";

export const PeersContext = createContext([]);
export const ThemeContext = createContext("light");
export const TransferContext = createContext(null);

export default function StateProvider({ children }) {
  // Periodically fetch list of peers from the backend
  const [peers, setPeers] = useState([]);
  useEffect(() => {
    EventsOn(PEERS_UPDATED, () => GetPeers().then((names) => setPeers(names)));
  }, []);

  // TODO: save to disk
  const defaultLight = window.matchMedia('(prefers-color-scheme: light)').matches;
  const [theme, setTheme] = useState(defaultLight ? "light" : "dark");

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
  }, [theme]);

  const [selectedPeers, setSelectedPeers] = useState([]);
  const [selectedFiles, setSelectedFiles] = useState([]);
  const [transferIds, setTransferIds] = useState([]);
  const [sending, setSending] = useState(false);

  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      <PeersContext.Provider value={peers}>
        <TransferContext.Provider
          value={{
            sending, setSending,
            transferIds, setTransferIds,
            selectedPeers, setSelectedPeers,
            selectedFiles, setSelectedFiles
          }}>
            {children}
        </TransferContext.Provider>
      </PeersContext.Provider>
    </ThemeContext.Provider>
  );
}
