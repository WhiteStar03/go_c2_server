import React, { useState } from 'react';
import { Routes, Route, Link, useNavigate } from 'react-router-dom';
import AuthForm from './components/AuthForm';
import Dashboard from './components/Dashboard';

function App() {
  const [token, setToken] = useState(localStorage.getItem('token'));
  const navigate = useNavigate();

  const logout = () => {
    setToken(null);
    localStorage.removeItem('token');
    navigate('/login');
  };

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
        <Route path="/login" element={<AuthForm isLogin={true} setToken={setToken} />} />
        <Route path="/register" element={<AuthForm isLogin={false} />} />
        <Route path="/dashboard" element={<Dashboard />} />
      </Routes>
    </div>
  );
}

export default App;