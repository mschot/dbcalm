import React from 'react';
import { useState } from 'react';
import { Api } from '../utils/api';
import { useProcessMonitor } from '../hooks/useProcessMonitor';

interface BackupActionMenuProps {
  backupId: string;
  onDelete?: () => void;
}

export const BackupActionMenu: React.FC<BackupActionMenuProps> = ({
  backupId,
  onDelete,
}) => {
  const [menuOpen, setMenuOpen] = useState<string | null>(null);
  const isOpen = menuOpen === backupId;
  const { startMonitoring, addToast } = useProcessMonitor();

  const handleCreateBackup = async (fromId: string) => {
    try {
      const response = await Api.post('/backups', {
        type: 'incremental',
        from_backup_id: fromId
      });
      startMonitoring(response, 'incremental_backup');
      setMenuOpen(null);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error';
      addToast(`Backup creation failed: ${message}`, 'error');
    }
  };

  const handleRestore = async (id: string, target: 'folder' | 'database') => {
    try {
      const response = await Api.post('/restore', {
        id,
        target
      });
      startMonitoring(response, 'restore');
      setMenuOpen(null);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error';
      addToast(`Restore failed: ${message}`, 'error');
    }
  };

  const handleDelete = async () => {
    const isConfirmed = window.confirm(
      'Are you sure you want to delete this backup? This action cannot be undone.'
    );

    if (isConfirmed) {
      try {
        await Api.delete(`/backups/${backupId}`);
        setMenuOpen(null);
        if (onDelete) {
          onDelete();
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Unknown error';
        addToast(`Delete failed: ${message}`, 'error');
      }
    }
  };

  return (
    <div className="dropdown dropdown-end">
      <button
        onClick={() => setMenuOpen(isOpen ? null : backupId)}
        className="btn btn-ghost btn-sm btn-circle"
      >
        <svg className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
          <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
        </svg>
      </button>

      {isOpen && (
        <ul className="dropdown-content menu menu-sm bg-base-200 rounded-box w-60 p-2 shadow-lg">
          <li>
            <button
              onClick={() => handleCreateBackup(backupId)}
              className="text-sm"
            >
              <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
              </svg>
              Create backup from here
            </button>
          </li>
          <li>
            <button
              onClick={() => handleRestore(backupId, 'folder')}
              className="text-sm"
            >
              <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
              </svg>
              Restore to folder
            </button>
          </li>
          <li>
            <button
              onClick={() => handleRestore(backupId, 'database')}
              className="text-sm"
            >
              <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
              </svg>
              Restore to database
            </button>
          </li>
          <li>
            <button
              onClick={handleDelete}
              className="text-sm text-error"
            >
              <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
              Delete
            </button>
          </li>
        </ul>
      )}
    </div>
  );
};
