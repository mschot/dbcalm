import { useState, useEffect } from 'react';
import { format } from 'date-fns';
import { Api } from '../utils/api';
import { RestoreTypeIcon } from '../components/RestoreTypeIcon';
import { RestoreResponse } from '../types/restoreResponse';
import { Header } from '../components/Header';
import { Pagination, PaginationResponse } from '../components/Pagination';
import { FilterBar } from '../components/FilterBar';

interface DisplayRestore {
  id: number;
  start_time: Date;
  end_time: Date | null;
  target: 'database' | 'folder';
  target_path: string;
  backup_timestamp: Date | null;
}

const Restores = () => {
  const [restores, setRestores] = useState<DisplayRestore[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [paginationResponse, setPaginationResponse] = useState<PaginationResponse>({
    total: 0,
    page: 1,
    per_page: 25,
    total_pages: 1
  });
  const [queryString, setQueryString] = useState<string>("");

  useEffect(() => {
    const fetchRestores = async () => {
      try {
        let queryParams = `order=start_time|desc&page=${currentPage}`;

        if (queryString) {
          queryParams += `&query=${queryString}`;
        }

        const response = await Api.get(`/restores?${queryParams}`) as RestoreResponse;
        const formattedRestores = response.items.map(item => ({
          id: item.id,
          start_time: new Date(item.start_time),
          end_time: item.end_time ? new Date(item.end_time) : null,
          target: item.target,
          target_path: item.target_path,
          backup_timestamp: item.backup_timestamp ? new Date(item.backup_timestamp) : null,
        }));
        setRestores(formattedRestores);
        setPaginationResponse(response.pagination);
      } catch (error) {
        console.error('Failed to fetch restores:', error);
      }
    };

    fetchRestores();
  }, [currentPage, queryString]);

  // Truncate path to show last 2 segments (e.g., /restores/2025-10-03-14-30-45)
  const getTruncatedPath = (path: string) => {
    const parts = path.split('/').filter(p => p);
    if (parts.length <= 2) return path;
    return '.../' + parts.slice(-2).join('/');
  };

  return (
    <div className="min-h-screen bg-base-200 p-4">
      <div className="container mx-auto">
        <Header currentPage="restores" />

        <div className="card bg-base-100 shadow-xl">
          <div className="card-body p-0">
            <FilterBar
              filters={[
                { type: 'time', fieldName: 'start_time' },
                {
                  type: 'select',
                  fieldName: 'target',
                  operator: 'eq',
                  options: [
                    { value: '', label: 'All types' },
                    { value: 'database', label: 'Database' },
                    { value: 'folder', label: 'Folder' }
                  ]
                }
              ]}
              onQueryChange={setQueryString}
            />
            <div>
              <table className="table table-zebra w-full">
                <thead>
                  <tr>
                    <th className="text-base-content">Type</th>
                    <th className="text-base-content">Restored</th>
                    <th className="text-base-content">Backup Created</th>
                    <th className="text-base-content">Target Path</th>
                  </tr>
                </thead>
                <tbody>
                  {restores.length === 0 ? (
                    <tr>
                      <td colSpan={4} className="text-center py-8 text-gray-500">
                        No restores found.
                      </td>
                    </tr>
                  ) : (
                    restores.map((restore) => (
                      <tr key={restore.id} className="hover">
                        <td>
                          <RestoreTypeIcon type={restore.target} />
                        </td>
                        <td>{format(restore.start_time, 'MMM d, yyyy HH:mm')}</td>
                        <td>
                          {restore.backup_timestamp ? (
                            format(restore.backup_timestamp, 'MMM d, yyyy HH:mm')
                          ) : (
                            <span className="text-gray-500 italic">N/A</span>
                          )}
                        </td>
                        <td>
                          {restore.target === 'folder' ? (
                            <div
                              className="group relative cursor-help"
                              title={restore.target_path}
                            >
                              <span>{getTruncatedPath(restore.target_path)}</span>
                              <span className="opacity-0 group-hover:opacity-100 absolute left-0 top-full mt-1 bg-base-200 text-sm px-2 py-1 rounded z-10 whitespace-nowrap">
                                {restore.target_path}
                              </span>
                            </div>
                          ) : (
                            <span className="text-gray-500 italic">Database</span>
                          )}
                        </td>
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

export default Restores;
