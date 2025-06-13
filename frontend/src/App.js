// c2client/src/App.js
import React, { useState, useEffect } from 'react'; // Added useEffect
import { Routes, Route, Link, useNavigate } from 'react-router-dom';
import AuthForm from './components/AuthForm';
import Dashboard from './components/Dashboard';

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

  return <p>Loading...</p>; 
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
        {/* Add a root route */}
        <Route path="/" element={<Home />} /> 
        <Route path="/login" element={<AuthForm isLogin={true} setToken={handleSetToken} />} />
        <Route path="/register" element={<AuthForm isLogin={false} setToken={handleSetToken} />} /> {/* Assuming register might also log in */}
        <Route path="/dashboard" element={<Dashboard token={token} />} /> {/* Pass token if Dashboard needs it directly */}
      </Routes>
    </div>
  );
}

export default App;