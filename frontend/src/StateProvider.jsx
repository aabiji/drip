import { createContext, useEffect, useState } from "react";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { GetPeers } from "../wailsjs/go/main/App";

export const PeersContext = createContext([]);
export const ThemeContext = createContext("light");

export default function StateProvider({ children }) {
  // Periodically fetch list of peers from the backend
  const [peers, setPeers] = useState([]);
  useEffect(() => {
    EventsOn("peers-updated", () => GetPeers().then((names) => setPeers(names)));
  }, []);

  // TODO: save to disk
  const defaultLight = window.matchMedia('(prefers-color-scheme: light)').matches;
  const [theme, setTheme] = useState(defaultLight ? "light" : "dark");

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
  }, [theme]);

  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      <PeersContext.Provider value={peers}>{children}</PeersContext.Provider>
    </ThemeContext.Provider>
  );
}
