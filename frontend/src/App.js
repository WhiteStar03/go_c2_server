// c2client/src/App.js
import React, { useState, useEffect } from 'react'; // Added useEffect
import { Routes, Route, Link, useNavigate } from 'react-router-dom';
import AuthForm from './components/AuthForm';
import Dashboard from './components/Dashboard';
import ProtectedRoute from './components/ProtectedRoute';
import PublicRoute from './components/PublicRoute';
import API_CONFIG from './config/api';

// Optional: Create a simple Home component or use Dashboard/Login
function Home() {
  const navigate = useNavigate();
  const token = localStorage.getItem('token');

  useEffect(() => {
    if (token) {
      navigate('/dashboard');
    } else {
      navigate('/login');
    }
  }, [navigate, token]);

  return <p>Loading...</p>; // Or some placeholder
}


function App() {
  const [token, setToken] = useState(localStorage.getItem('token'));
  const navigate = useNavigate();

  // Check token validity on app load
  useEffect(() => {
    const checkTokenValidity = async () => {
      const storedToken = localStorage.getItem('token');
      if (storedToken) {
        try {
          // Test the token by making a request to a protected endpoint
          const response = await fetch(`${API_CONFIG.API_BASE}/implants`, {
            headers: { Authorization: `Bearer ${storedToken}` }
          });
          
          if (!response.ok) {
            // Token is invalid or expired
            localStorage.removeItem('token');
            setToken(null);
            if (window.location.pathname !== '/login' && window.location.pathname !== '/register') {
              navigate('/login');
            }
          } else {
            setToken(storedToken);
          }
        } catch (error) {
          console.error('Token validation error:', error);
          localStorage.removeItem('token');
          setToken(null);
        }
      }
    };

    checkTokenValidity();
  }, [navigate]);

  const handleSetToken = (newToken) => {
    setToken(newToken);
    if (newToken) {
      localStorage.setItem('token', newToken);
    } else {
      localStorage.removeItem('token');
    }
  };
  
  const logout = () => {
    handleSetToken(null);
    navigate('/login');
  };

  // If you want to protect routes, you'd typically wrap them
  // or check token in a useEffect within Dashboard.
  // For simplicity, this example keeps Dashboard accessible but it would fetch data using the token.

  return (
    <div className="min-h-screen bg-gray-900 text-white">
      <nav className="p-4 bg-gray-800 flex justify-between shadow-lg">
        <Link to="/" className="text-lg font-bold text-white">C2 Panel</Link>
        
        <div className="flex space-x-4">
          {token ? (
            <>
              <Link to="/dashboard" className="hover:underline text-white">Dashboard</Link>
              <button onClick={logout} className="hover:underline text-white">Logout</button>
            </>
          ) : (
            <>
              <Link to="/login" className="hover:underline text-white">Login</Link>
              <Link to="/register" className="hover:underline text-white">Register</Link>
            </>
          )}
        </div>
      </nav>
      <Routes>
        <Route path="/" element={<Home />} /> 
        <Route path="/login" element={
          <PublicRoute token={token}>
            <AuthForm isLogin={true} setToken={handleSetToken} />
          </PublicRoute>
        } />
        <Route path="/register" element={
          <PublicRoute token={token}>
            <AuthForm isLogin={false} setToken={handleSetToken} />
          </PublicRoute>
        } />
        <Route path="/dashboard" element={
          <ProtectedRoute token={token}>
            <Dashboard token={token} />
          </ProtectedRoute>
        } />
      </Routes>
    </div>
  );
}

export default App;