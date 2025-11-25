import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Api } from "../utils/api";
import { useProcessMonitor } from "../hooks/useProcessMonitor";
import { useTheme } from "../hooks/useTheme";

interface HeaderProps {
  currentPage?: "backups" | "clients" | "processes" | "restores" | "schedules";
}

export const Header = ({ currentPage = "backups" }: HeaderProps) => {
  const [createMenuOpen, setCreateMenuOpen] = useState(false);
  const [burgerMenuOpen, setBurgerMenuOpen] = useState(false);
  const [showRestoreInfo, setShowRestoreInfo] = useState(false);
  const navigate = useNavigate();
  const { startMonitoring, activeProcesses, addToast } = useProcessMonitor();
  const { theme, toggleTheme } = useTheme();

  const handleLogout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('tokenExpiry');
    window.location.href = '/login';
  };

  const handleNewBackup = async (type: 'full' | 'incremental') => {
    try {
      const response = await Api.post('/backups', { type });
      const processType = type === 'full' ? 'full_backup' : 'incremental_backup';
      startMonitoring(response, processType);
      setCreateMenuOpen(false);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error';
      addToast(`Backup creation failed: ${message}`, 'error');
    }
  };

  const handleAddClient = () => {
    navigate('/add-client');
    setCreateMenuOpen(false);
  };

  const handleAddSchedule = () => {
    navigate('/add-schedule');
    setCreateMenuOpen(false);
  };

  const handleAddClick = (currentPage: string) => {
    if (currentPage === "backups") {
      setCreateMenuOpen(!createMenuOpen);
    } else if (currentPage === "clients") {
      handleAddClient();
    } else if (currentPage === "schedules") {
      handleAddSchedule();
    } else if (currentPage === "restores") {
      setShowRestoreInfo(true);
    }
  };

  return (
    <>
      {showRestoreInfo && (
        <div className="modal modal-open">
          <div className="modal-box">
            <h3 className="font-bold text-lg">How to Restore a Backup</h3>
            <p className="py-4">
              To restore a backup, go to the <a href="/dashboard" className="link link-primary">Backups page</a> and click the actions menu (⋮) on the backup you want to restore.
            </p>
            <div className="modal-action">
              <button onClick={() => setShowRestoreInfo(false)} className="btn">
                Got it
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="flex justify-between items-center mb-6">
        <div className="flex items-center gap-6 mt-2">
          <h1 className="text-3xl font-bold text-primary">
            {currentPage === "backups" ? "Backups"
             : currentPage === "clients" ? "API keys"
             : currentPage === "schedules" ? "Schedules"
             : currentPage === "processes" ? "Processes"
             : "Restores"}
          </h1>
        </div>
      <div className="flex flex-col items-end gap-2">
        <div className="flex items-center gap-2">
          {activeProcesses.length > 0 && (
            <div className="flex items-center gap-2 text-sm opacity-70">
              <span className="loading loading-spinner loading-sm"></span>
              <span>{activeProcesses.length} process{activeProcesses.length > 1 ? 'es' : ''} running</span>
            </div>
          )}

          {currentPage !== "processes" && (
            <div className="dropdown dropdown-end">
              <button
                onClick={() => handleAddClick(currentPage)}
                className="btn btn-circle btn-sm"
              >
                <svg
                  className="h-5 w-5"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 4v16m8-8H4"
                  />
                </svg>
              </button>

            {createMenuOpen && (
              <ul className="dropdown-content menu menu-sm bg-base-200 rounded-box w-48 p-2 shadow-lg">
                {currentPage === "backups" ? (
                  <>
                    <li>
                      <button
                        onClick={() => handleNewBackup('full')}
                        className="text-sm"
                      >
                        New full backup
                      </button>
                    </li>
                    <li>
                      <button
                        onClick={() => handleNewBackup('incremental')}
                        className="text-sm"
                      >
                        New incremental backup
                      </button>
                    </li>
                  </>
                ): null}
              </ul>
            )}
            </div>
          )}

          <button
            onClick={toggleTheme}
            className="btn btn-ghost btn-sm opacity-50 hover:opacity-100 transition-opacity"
            aria-label="Toggle theme"
          >
            {theme === 'light' ? (
              <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
              </svg>
            ) : (
              <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
              </svg>
            )}
          </button>

          <div className="dropdown dropdown-end">
            <button
              onClick={() => setBurgerMenuOpen(!burgerMenuOpen)}
              className="btn btn-ghost btn-sm opacity-50 hover:opacity-100 transition-opacity"
            >
              <svg
                className="h-5 w-5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 6h16M4 12h16M4 18h16"
                />
              </svg>
            </button>

          {burgerMenuOpen && (
            <ul className="dropdown-content menu menu-sm bg-base-200 rounded-box w-48 p-2 shadow-lg">
              <li>
                <a href="/dashboard" className="text-sm">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
                  </svg>
                  Backups
                </a>
              </li>
              <li>
                <a href="/restores" className="text-sm">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                  Restores
                </a>
              </li>
              <li>
                <a href="/schedules" className="text-sm">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                  </svg>
                  Schedules
                </a>
              </li>
              <li>
                <a href="/clients" className="text-sm">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
                  </svg>
                  API keys
                </a>
              </li>
              <li>
                <a href="/processes" className="text-sm">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  </svg>
                  Processes
                </a>
              </li>
              <li>
                <button
                  onClick={handleLogout}
                  className="text-sm"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                  </svg>
                  Log out
                </button>
              </li>
            </ul>
          )}
          </div>
        </div>
      </div>
    </div>
    </>
  );
};
