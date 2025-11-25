import { useContext } from "react";
import { ProcessMonitorContext } from "../contexts/ProcessMonitorContext";

export const useProcessMonitor = () => {
  const context = useContext(ProcessMonitorContext);

  if (!context) {
    throw new Error("useProcessMonitor must be used within ProcessMonitorProvider");
  }

  return context;
};
