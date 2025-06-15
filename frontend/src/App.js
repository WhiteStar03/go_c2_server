
import React, { useState, useEffect } from 'react'; 
import { Routes, Route, Link, useNavigate } from 'react-router-dom';
import AuthForm from './components/AuthForm';
import Dashboard from './components/Dashboard';
import ProtectedRoute from './components/ProtectedRoute';

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

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-900">
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto mb-4"></div>
        <h2 className="text-2xl font-bold text-white mb-2">Welcome to C2 Panel</h2>
        <p className="text-gray-400 text-lg">Please login or register into your account</p>
      </div>
    </div>
  ); 
}


function App() {
  const [token, setToken] = useState(localStorage.getItem('token'));
  const navigate = useNavigate();

  const handleSetToken = (newToken) => {
    setToken(newToken);
    if (newToken) {
      localStorage.setItem('token', newToken);
      navigate('/dashboard'); 
    } else {
      localStorage.removeItem('token');
    }
  };
  
  const logout = () => {
    handleSetToken(null); 
    navigate('/');
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
        {/* main routes setup here */}
        <Route path="/" element={<Home />} /> 
        <Route path="/login" element={<AuthForm isLogin={true} setToken={handleSetToken} />} />
        <Route path="/register" element={<AuthForm isLogin={false} setToken={handleSetToken} />} /> {/* registration flow in case needed */}
        <Route path="/dashboard" element={
          <ProtectedRoute>
            <Dashboard token={token} />
          </ProtectedRoute>
        } /> {/* dashboard protected from unauthorized access */}
      </Routes>
    </div>
  );
}

export default App;