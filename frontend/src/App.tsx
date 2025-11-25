import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Clients from './pages/Clients';
import Processes from './pages/Processes';
import Restores from './pages/Restores';
import Schedules from './pages/Schedules';
import { checkAuth } from './utils/auth';
import AddClient from './pages/AddClient';
import ScheduleForm from './pages/ScheduleForm';
import { ProcessMonitorProvider } from './contexts/ProcessMonitorContext';
import { ThemeProvider } from './contexts/ThemeContext';
import { Toast } from './components/Toast';
import { useProcessMonitor } from './hooks/useProcessMonitor';
import './App.css'

const ToastContainer = () => {
  const { toastMessages, dismissToast } = useProcessMonitor();
  return <Toast messages={toastMessages} onDismiss={dismissToast} />;
};

const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  const isAuthenticated = checkAuth();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};

function App() {
  return (
    <ThemeProvider>
      <Router>
        <ProcessMonitorProvider>
          <ToastContainer />
          <Routes>
          <Route path="/login" element={<Login />} />
          <Route
            path="/dashboard"
            element={
              <ProtectedRoute>
                <Dashboard />
              </ProtectedRoute>
            }
          />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <Dashboard />
              </ProtectedRoute>
            }
          />
          <Route
            path="/clients"
            element={
              <ProtectedRoute>
                <Clients />
              </ProtectedRoute>
            }
          />
          <Route
           path="/add-client"
           element={
            <ProtectedRoute>
              <AddClient />
            </ProtectedRoute>
            }
          />
          <Route
            path="/processes"
            element={
              <ProtectedRoute>
                <Processes />
              </ProtectedRoute>
            }
          />
          <Route
            path="/restores"
            element={
              <ProtectedRoute>
                <Restores />
              </ProtectedRoute>
            }
          />
          <Route
            path="/schedules"
            element={
              <ProtectedRoute>
                <Schedules />
              </ProtectedRoute>
            }
          />
          <Route
            path="/add-schedule"
            element={
              <ProtectedRoute>
                <ScheduleForm />
              </ProtectedRoute>
            }
          />
          <Route
            path="/schedules/edit/:id"
            element={
              <ProtectedRoute>
                <ScheduleForm />
              </ProtectedRoute>
            }
          />
          </Routes>
        </ProcessMonitorProvider>
      </Router>
    </ThemeProvider>
  );
}

export default App;
