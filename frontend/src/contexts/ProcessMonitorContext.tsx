import { createContext, useCallback, useEffect, useState, ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { Api } from "../utils/api";
import { ToastMessage } from "../components/Toast";

export interface ActiveProcess {
  pid: string;
  type: "full_backup" | "incremental_backup" | "restore";
  statusLink: string;
  startTime: Date;
  status: "running" | "success" | "failed";
}

interface ProcessMonitorContextType {
  activeProcesses: ActiveProcess[];
  startMonitoring: (response: { pid: string; link: string; status: string }, type: ActiveProcess["type"]) => void;
  stopMonitoring: (pid: string) => void;
  toastMessages: ToastMessage[];
  dismissToast: (id: string) => void;
  addToast: (message: string, type: ToastMessage["type"], link?: string, linkText?: string) => void;
}

export const ProcessMonitorContext = createContext<ProcessMonitorContextType | undefined>(undefined);

interface ProcessMonitorProviderProps {
  children: ReactNode;
}

export const ProcessMonitorProvider = ({ children }: ProcessMonitorProviderProps) => {
  const [activeProcesses, setActiveProcesses] = useState<ActiveProcess[]>([]);
  const [toastMessages, setToastMessages] = useState<ToastMessage[]>([]);
  const navigate = useNavigate();

  const addToast = useCallback((
    message: string,
    type: ToastMessage["type"],
    link?: string,
    linkText?: string
  ) => {
    const id = `${Date.now()}-${Math.random()}`;
    setToastMessages((prev) => [...prev, { id, message, type, link, linkText }]);
  }, []);

  const dismissToast = useCallback((id: string) => {
    setToastMessages((prev) => prev.filter((msg) => msg.id !== id));
  }, []);

  const startMonitoring = useCallback((
    response: { pid: string; link: string; status: string },
    type: ActiveProcess["type"]
  ) => {
    const process: ActiveProcess = {
      pid: response.pid,
      type,
      statusLink: response.link,
      startTime: new Date(),
      status: "running",
    };

    setActiveProcesses((prev) => [...prev, process]);

    const typeLabel = type.replace("_", " ");
    addToast(`Starting ${typeLabel}...`, "info");
  }, [addToast]);

  const stopMonitoring = useCallback((pid: string) => {
    setActiveProcesses((prev) => prev.filter((p) => p.pid !== pid));
  }, []);

  const checkProcessStatus = useCallback(async (process: ActiveProcess) => {
    try {
      const response = await Api.get(process.statusLink);

      if (response.status !== "running") {
        const typeLabel = process.type.replace("_", " ");

        if (response.status === "success") {
          addToast(`${typeLabel} completed successfully`, "success");
        } else if (response.status === "failed") {
          addToast(
            `${typeLabel} failed`,
            "error",
            `/processes?filter=${process.pid}`,
            "View Details"
          );
        }

        stopMonitoring(process.pid);

        // Navigate to appropriate page based on process type
        if (response.status === "success") {
          setTimeout(() => {
            if (process.type === "restore") {
              navigate("/restores");
            } else {
              // For backups, refresh current page
              window.location.reload();
            }
          }, 1000);
        }
      }
    } catch (error) {
      console.error(`Failed to check status for process ${process.pid}:`, error);
      const typeLabel = process.type.replace("_", " ");
      addToast(`Failed to check status for ${typeLabel}`, "error");
      stopMonitoring(process.pid);
    }
  }, [addToast, stopMonitoring, navigate]);

  useEffect(() => {
    if (activeProcesses.length === 0) return;

    const interval = setInterval(() => {
      activeProcesses.forEach((process) => {
        checkProcessStatus(process);
      });
    }, 10000); // Poll every 10 seconds

    // Also check immediately
    activeProcesses.forEach((process) => {
      checkProcessStatus(process);
    });

    return () => clearInterval(interval);
  }, [activeProcesses, checkProcessStatus]);

  return (
    <ProcessMonitorContext.Provider
      value={{
        activeProcesses,
        startMonitoring,
        stopMonitoring,
        toastMessages,
        dismissToast,
        addToast,
      }}
    >
      {children}
    </ProcessMonitorContext.Provider>
  );
};
