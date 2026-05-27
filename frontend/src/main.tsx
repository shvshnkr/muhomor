import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import { ConnectionProvider } from "./stores/connection";
import { ShutdownProvider } from "./stores/shutdown";
import "./styles/globals.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ShutdownProvider>
      <ConnectionProvider>
        <App />
      </ConnectionProvider>
    </ShutdownProvider>
  </React.StrictMode>
);
