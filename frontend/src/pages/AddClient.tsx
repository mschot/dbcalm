import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Api } from '../utils/api';
import { Header } from '../components/Header';

interface ClientCredentials {
  id: string;
  secret: string;
  scopes: string[];
  label: string;
}

const AddClient = () => {
  const [label, setLabel] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [credentials, setCredentials] = useState<ClientCredentials | null>(null);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!label.trim()) {
      setError('API key label is required');
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      const response = await Api.post('/clients', { label }) as ClientCredentials;
      setCredentials(response);
    } catch (err) {
      console.error('Failed to create API key:', err);
      setError('Failed to create API key. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };

  if (credentials) {
    return (
    <div className="min-h-screen bg-base-200 p-4">
      <div className="container mx-auto">
        <Header currentPage="clients" />

        <div className="card bg-base-100 shadow-xl">
          <div className="card-body">
              <h2 className="card-title text-2xl mb-2">API Key Created Successfully!</h2>

              <div className="alert alert-warning mb-6">
                <svg xmlns="http://www.w3.org/2000/svg" className="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                </svg>
                <span>
                  <strong>Important:</strong> Please copy and store the API secret safely.
                  You won't be able to see it again after leaving this page.
                </span>
              </div>

              <div className="bg-base-200 p-4 rounded-lg mb-4">
                <div className="mb-4">
                  <label className="text-sm font-semibold">Label:</label>
                  <div className="font-mono bg-base-100 p-2 rounded mt-1">
                    {credentials.label}
                  </div>
                </div>

                <div className="mb-4">
                  <label className="text-sm font-semibold">API Key ID:</label>
                  <div className="font-mono bg-base-100 p-2 rounded mt-1">
                    {credentials.id}
                  </div>
                </div>

                <div className="mb-4">
                  <label className="text-sm font-semibold">API Secret:</label>
                  <div className="font-mono bg-base-100 p-2 rounded mt-1">
                    {credentials.secret}
              </div>
                </div>
            </div>

              <div className="card-actions justify-end">
                <button
                  className="btn btn-primary"
                  onClick={() => navigate('/clients')}
                >
                  Continue to API keys
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
  );
  }

  return (
    <div className="min-h-screen bg-base-200 p-4">
      <div className="container mx-auto">
        <Header currentPage="clients" />

        <div className="card bg-base-100 shadow-xl">
          <div className="card-body">
            <h2 className="card-title text-2xl mb-6">Add New API Key</h2>

            {error && (
              <div className="alert alert-error mb-4">
                <svg xmlns="http://www.w3.org/2000/svg" className="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span>{error}</span>
              </div>
            )}

            <form onSubmit={handleSubmit}>
              <div className="form-control">
                <input
                  type="text"
                  placeholder="Enter API key label"
                  className="input input-bordered"
                  value={label}
                  onChange={(e) => setLabel(e.target.value)}
                  disabled={isSubmitting}
                />
              </div>

              <div className="form-control mt-6">
                <button
                  type="submit"
                  className={`btn btn-primary ${isSubmitting ? 'loading' : ''}`}
                  disabled={isSubmitting}
                >
                  {isSubmitting ? 'Creating...' : 'Create API Key'}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AddClient;
