import React, { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

function ProtectedRoute({ children, token }) {
  const navigate = useNavigate();

  useEffect(() => {
    if (!token) {
      navigate('/login');
    }
  }, [token, navigate]);

  // If no token, don't render children (will redirect)
  if (!token) {
    return <div className="min-h-screen bg-gray-900 text-white flex items-center justify-center">
      <p>Redirecting to login...</p>
    </div>;
  }

  return children;
}

export default ProtectedRoute;
