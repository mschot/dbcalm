import { useState, FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { handleLogin } from '../actions/login';

const Login = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const navigate = useNavigate();
  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    const result = await handleLogin(username, password);
    setLoading(false);
    if(!result.success) {
      setError(result.error || 'Error logging in');
      return
    }
    navigate('/dashboard')
  };

  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="card w-96 bg-base-100 shadow-xl">
        <div className="card-body">
          <div className="flex justify-center mb-4">
            <img src="/logo.png" alt="DBCalm Logo" className="h-16" />
          </div>
          <h2 className="card-title">Login</h2>
          {error && (
            <div className="text-red-500 text-sm text-center">
              {error}
            </div>
          )}
          <form onSubmit={handleSubmit}>
            <div className="form-control w-full">
              <label className="label">
                <span className="label-text">Username</span>
              </label>
              <input
                type="text"
                className="input"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />
            </div>
            <div className="form-control w-full">
              <label className="label">
                <span className="label-text">Password</span>
              </label>
              <input
                type="password"
                className="input"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            <div className="card-actions justify-end mt-6">
              <button type="submit" className="btn btn-primary" disabled={loading}>Login</button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};

export default Login;