import { useState, useEffect } from 'react';
import { format } from 'date-fns';
import { Api } from '../utils/api';
import { BackupTypeIcon } from '../components/BackupTypeIcon';
import { Backup } from '../types/backup';
import { BackupResponse } from '../types/backupResponse';
import { Header } from '../components/Header';
import { Pagination, PaginationResponse } from '../components/Pagination';
import { BackupActionMenu } from '../components/BackupActionMenu';
import { FilterBar } from '../components/FilterBar';

const Dashboard = () => {
  const [backups, setBackups] = useState<Backup[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [paginationResponse, setPaginationResponse] = useState<PaginationResponse>({
    total: 0,
    page: 1,
    per_page: 25,
    total_pages: 1
  });
  const [queryString, setQueryString] = useState<string>("");
  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedBackups, setSelectedBackups] = useState<Set<string>>(new Set());

  const fetchBackups = async () => {
    try {
      let queryParams = `order=start_time|desc&page=${currentPage}`;

      if (queryString) {
        queryParams += `&query=${queryString}`;
      }

      const response = await Api.get(`/backups?${queryParams}`) as BackupResponse;
      const formattedBackups = response.items.map(item => ({
        id: item.id,
        type: !item.from_backup_id ? 'full' : 'incremental',
        created: new Date(item.start_time),
        retentionValue: item.retention_value,
        retentionUnit: item.retention_unit
      }));
      setBackups(formattedBackups);
      setPaginationResponse(response.pagination);
    } catch (error) {
      console.error('Failed to fetch backups:', error);
    }
  };

  useEffect(() => {
    fetchBackups();
  }, [currentPage, queryString]);

  // Clear selection when changing pages
  useEffect(() => {
    setSelectedBackups(new Set());
  }, [currentPage]);

  const toggleSelectionMode = () => {
    setSelectionMode(!selectionMode);
    setSelectedBackups(new Set());
  };

  const toggleBackupSelection = (backupId: string) => {
    const newSelection = new Set(selectedBackups);
    if (newSelection.has(backupId)) {
      newSelection.delete(backupId);
    } else {
      newSelection.add(backupId);
    }
    setSelectedBackups(newSelection);
  };

  const toggleSelectAll = () => {
    if (selectedBackups.size === backups.length) {
      setSelectedBackups(new Set());
    } else {
      setSelectedBackups(new Set(backups.map(b => b.id)));
    }
  };

  const formatRetention = (backup: Backup): string => {
    if (backup.retentionValue && backup.retentionUnit) {
      return `${backup.retentionValue} ${backup.retentionUnit}`;
    }
    return 'Forever';
  };

  const handleBulkDelete = async () => {
    const count = selectedBackups.size;
    const isConfirmed = window.confirm(
      `Are you sure you want to delete ${count} backup${count > 1 ? 's' : ''}? This action cannot be undone.`
    );

    if (isConfirmed) {
      try {
        // Delete all selected backups
        await Promise.all(
          Array.from(selectedBackups).map(id => Api.delete(`/backups/${id}`))
        );

        // Refresh the backup list
        const queryParams = queryString
          ? `order=start_time|desc&page=${currentPage}&query=${queryString}`
          : `order=start_time|desc&page=${currentPage}`;
        const response = await Api.get(`/backups?${queryParams}`) as BackupResponse;
        const formattedBackups = response.items.map(item => ({
          id: item.id,
          type: !item.from_backup_id ? 'full' : 'incremental',
          created: new Date(item.start_time),
          retentionValue: item.retention_value,
          retentionUnit: item.retention_unit
        }));
        setBackups(formattedBackups);
        setPaginationResponse(response.pagination);

        // Exit selection mode
        setSelectionMode(false);
        setSelectedBackups(new Set());
      } catch (error) {
        console.error('Failed to delete backups:', error);
        alert('Failed to delete some backups. Please try again.');
      }
    }
  };

  return (
    <div className="min-h-screen bg-base-200 p-4">
      <div className="container mx-auto">
        <Header />

        <div className="card bg-base-100 shadow-xl">
          <div className="card-body p-0">
            <FilterBar
              filters={[
                { type: 'time', fieldName: 'start_time' },
                {
                  type: 'select',
                  fieldName: 'from_backup_id',
                  operator: 'eq',
                  options: [
                    { value: '', label: 'All types' },
                    { value: 'null', label: 'Full' },
                    { value: 'not_null', label: 'Incremental' }
                  ]
                }
              ]}
              onQueryChange={setQueryString}
              actionButtons={
                <>
                  {selectionMode && (
                    <button
                      onClick={handleBulkDelete}
                      disabled={selectedBackups.size === 0}
                      className={`btn btn-sm ${selectedBackups.size === 0 ? 'btn-disabled opacity-50' : 'btn-error'}`}
                      title="Delete selected backups"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  )}
                  <button
                    onClick={toggleSelectionMode}
                    className={`btn btn-sm ${selectionMode ? 'btn-ghost' : 'btn-outline'}`}
                  >
                    {selectionMode ? 'Cancel Selection' : 'Select'}
                  </button>
                </>
              }
            />

            {/* Selection count - always takes up space to prevent table shift */}
            <div className="px-4 pt-2 text-sm min-h-[1.5rem] flex items-center" style={{ opacity: selectionMode && selectedBackups.size > 0 ? 1 : 0 }}>
              {selectionMode && selectedBackups.size > 0 ? `${selectedBackups.size} selected` : '\u00A0'}
            </div>

            <div>
              <table className="table table-zebra w-full">
                <thead>
                  <tr>
                    {selectionMode && (
                      <th className="text-base-content w-12">
                        <input
                          type="checkbox"
                          className="checkbox checkbox-sm"
                          checked={backups.length > 0 && selectedBackups.size === backups.length}
                          onChange={toggleSelectAll}
                        />
                      </th>
                    )}
                    <th className="text-base-content">Type</th>
                    <th className="text-base-content">Created</th>
                    <th className="text-base-content">Retention</th>
                    {!selectionMode && <th className="text-right text-base-content">Actions</th>}
                  </tr>
                </thead>
                <tbody>
                  {backups.length === 0 ? (
                    <tr>
                      <td colSpan={selectionMode ? 4 : 4} className="text-center py-8 text-gray-500">
                        No backups found. Click the + button to create one.
                      </td>
                    </tr>
                  ) : (
                    backups.map((backup) => (
                      <tr key={backup.id} className="hover">
                        {selectionMode && (
                          <td>
                            <input
                              type="checkbox"
                              className="checkbox checkbox-sm"
                              checked={selectedBackups.has(backup.id)}
                              onChange={() => toggleBackupSelection(backup.id)}
                            />
                          </td>
                        )}
                        <td>
                          <BackupTypeIcon type={backup.type} />
                        </td>
                        <td>{format(backup.created, 'MMM d, yyyy HH:mm')}</td>
                        <td>{formatRetention(backup)}</td>
                        {!selectionMode && (
                          <td className="text-right">
                            <BackupActionMenu
                              backupId={backup.id}
                              onDelete={fetchBackups}
                            />
                          </td>
                        )}
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
            <Pagination
              paginationResponse={paginationResponse}
              onPageChange={setCurrentPage}
            />
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;