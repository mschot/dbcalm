import { useState, useEffect } from 'react';
import { Api } from '../utils/api';
import { Client } from '../types/client';
import { ClientResponse } from '../types/clientResponse';
import { Header } from '../components/Header';
import { Pagination, PaginationResponse } from '../components/Pagination';

const Clients = () => {
  const [clients, setClients] = useState<Client[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [paginationResponse, setPaginationResponse] = useState<PaginationResponse>({
    total: 0,
    page: 1,
    per_page: 25,
    total_pages: 1
  });
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editLabel, setEditLabel] = useState<string>('');

  useEffect(() => {
    const fetchClients = async () => {
      try {
        const response = await Api.get(`/clients?page=${currentPage}`) as ClientResponse;
        const formattedClients = response.items.map(item => ({
          id: item.id,
          label: item.label
        }));
        setClients(formattedClients);
        setPaginationResponse(response.pagination);
      } catch (error) {
        console.error('Failed to fetch clients:', error);
      }
    };

    fetchClients();
  }, [currentPage]);

  const handleDeleteClient = async (id: string) => {
    const isConfirmed = window.confirm(
      "Are you sure you want to delete this API key? This action cannot be undone and may affect services using this API key."
    );

    if (isConfirmed) {
      try {
        await Api.delete(`/clients/${id}`);
        // Refresh the clients list after deletion
        setClients(clients.filter(client => client.id !== id));
      } catch (error) {
        console.error('Failed to delete API key:', error);
        alert('Failed to delete the API key. Please try again.');
      }
    }
  };

  const startEditing = (client: Client) => {
    setEditingId(client.id);
    setEditLabel(client.label);
  };

  const cancelEditing = () => {
    setEditingId(null);
    setEditLabel('');
  };

  const saveClientLabel = async (id: string) => {
    try {
      await Api.put(`/clients/${id}`, { label: editLabel });

      // Update client in the local state
      setClients(clients.map(client =>
        client.id === id ? { ...client, label: editLabel } : client
      ));

      // Exit edit mode
      setEditingId(null);
    } catch (error) {
      console.error('Failed to update API key label:', error);
      alert('Failed to update the API key label. Please try again.');
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent, id: string) => {
    if (e.key === 'Enter') {
      saveClientLabel(id);
    } else if (e.key === 'Escape') {
      cancelEditing();
    }
  };

  return (
    <div className="min-h-screen bg-base-200 p-4">
      <div className="container mx-auto">
        <Header currentPage="clients" />

        <div className="card bg-base-100 shadow-xl">
          <div className="card-body p-0">
            <div>
              <table className="table table-zebra w-full">
                <thead>
                  <tr>
                    <th className="text-base-content">Label</th>
                    <th className="text-base-content">API Key ID</th>
                    <th className="text-right text-base-content"></th>
                  </tr>
                </thead>
                <tbody>
                  {clients.length === 0 ? (
                    <tr>
                      <td colSpan={3} className="text-center py-8 text-gray-500">
                        No API keys found. Click the + button to add one.
                      </td>
                    </tr>
                  ) : (
                    clients.map((client) => (
                      <tr key={client.id} className="hover">
                        <td className="font-medium">
                          {editingId === client.id ? (
                            <div className="flex items-center">
                              <input
                                type="text"
                                value={editLabel}
                                onChange={(e) => setEditLabel(e.target.value)}
                                onKeyDown={(e) => handleKeyDown(e, client.id)}
                                className="input input-bordered input-sm w-full max-w-xs"
                                autoFocus
                              />
                              <div className="flex gap-1 ml-2">
                                <button
                                  onClick={() => saveClientLabel(client.id)}
                                  className="btn btn-ghost btn-xs text-success"
                                >
                                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                                  </svg>
                                </button>
                                <button
                                  onClick={cancelEditing}
                                  className="btn btn-ghost btn-xs text-error"
                                >
                                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                  </svg>
                                </button>
                              </div>
                            </div>
                          ) : (
                            <div className="flex items-center">
                              {client.label}
                              <button
                                onClick={() => startEditing(client)}
                                className="btn btn-ghost btn-xs ml-2"
                                title="Edit label"
                              >
                                <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                                </svg>
                              </button>
                            </div>
                          )}
                        </td>
                        <td>{client.id}</td>
                        <td className="text-right">
                          <button
                            onClick={() => handleDeleteClient(client.id)}
                            className="btn btn-ghost btn-sm text-error"
                            title="Delete API key"
                          >
                            <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                            </svg>
                          </button>
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

export default Clients;