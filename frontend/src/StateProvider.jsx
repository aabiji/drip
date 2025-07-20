import { createContext, useEffect, useState } from "react";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { GetPeers } from "../wailsjs/go/main/App";
import { PEERS_UPDATED } from "./constants";

export const PeersContext = createContext([]);
export const ErrorContext = createContext([]);
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

  const [errors, setErrors] = useState([]);

  const addError = (message) =>
    setErrors(prev => [...prev, { id: crypto.randomUUID(), message }]);

  const removeError = (id) =>
    setErrors(prev => prev.filter(error => error.id !== id));

  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      <ErrorContext.Provider value={{ errors, addError, removeError }}>
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
      </ErrorContext.Provider>
    </ThemeContext.Provider>
  );
}
