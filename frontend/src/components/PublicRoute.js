import React, { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

function PublicRoute({ children, token }) {
  const navigate = useNavigate();

  useEffect(() => {
    // If user is authenticated, redirect to dashboard
    if (token) {
      navigate('/dashboard');
    }
  }, [token, navigate]);

  // If token exists, don't render children (will redirect)
  if (token) {
    return <div className="min-h-screen bg-gray-900 text-white flex items-center justify-center">
      <p>Redirecting to dashboard...</p>
    </div>;
  }

  return children;
}

export default PublicRoute;
