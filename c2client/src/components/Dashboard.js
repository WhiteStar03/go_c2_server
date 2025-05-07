import React, { useEffect, useState } from "react";
import Terminal from "./Terminal";

const API_BASE = "http://localhost:8080/api";
const REFRESH_INTERVAL = 7000; // Refresh implant list every 7 seconds

function Dashboard() {
  const [implants, setImplants] = useState([]);
  const [selectedImplant, setSelectedImplant] = useState(null); // Stores unique_token

  const fetchImplants = async () => {
    const token = localStorage.getItem("token");
    try {
      const res = await fetch(`${API_BASE}/implants`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        console.error("Error fetching implants:", res.status, errorData.error || res.statusText);
        // Potentially handle token expiry or other auth errors by logging out user
        if (res.status === 401) {
            alert("Session expired. Please log in again.");
            // localStorage.removeItem("token"); // or trigger logout function
            // window.location.href = "/login";
        }
        // setImplants([]); // Keep existing list on transient error or clear based on policy
        return;
      }
      const data = await res.json();
      setImplants(data.implants || []);
    } catch (err) {
      console.error("Network or other error fetching implants:", err);
      // setImplants([]);
    }
  };

  useEffect(() => {
    fetchImplants(); // Initial fetch
    const intervalId = setInterval(fetchImplants, REFRESH_INTERVAL);
    return () => clearInterval(intervalId); // Cleanup on unmount
  }, []);


  const generateImplant = async () => {
    const token = localStorage.getItem("token");
    try {
      const res = await fetch(`${API_BASE}/generate-implant`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
      });
      // This endpoint directly returns the binary file
      if (res.ok) {
        alert("Implant generated ");
      } else {
        const data = await res.json().catch(() => ({})); // Try to parse error JSON
        alert("Failed to generate implant: " + (data.error || res.statusText));
      }
    } catch (err) {
      console.error("Error generating implant:", err);
      alert("Failed to generate implant. See console for details.");
    }
  };

  const deleteImplant = async (uniqueToken) => {
    const token = localStorage.getItem("token");
    if (!window.confirm(`Are you sure you want to delete implant ${uniqueToken}? This action cannot be undone.`)) {
      return;
    }
    try {
      const res = await fetch(`${API_BASE}/implants/${uniqueToken}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        alert("Implant deleted successfully");
        fetchImplants();
        if (selectedImplant === uniqueToken) {
            setSelectedImplant(null); 
        }
      } else {
        const data = await res.json().catch(() => ({}));
        alert("Failed to delete implant: " + (data.error || res.statusText));
      }
    } catch (err) {
      console.error("Error deleting implant:", err);
      alert("Failed to delete implant. See console for details.");
    }
  };

  const sendQuickCommand = async (uniqueToken) => {
    const token = localStorage.getItem("token");
    const command = prompt("Enter command to send (for interactive session, use 'Terminal'):");
    if (!command) return;
    try {
      const res = await fetch(`${API_BASE}/send-command`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          implant_id: uniqueToken,
          command,
        }),
      });
      if (res.ok) {
        alert("Command sent successfully. Check implant logs or open terminal for output.");
      } else {
        const data = await res.json().catch(() => ({}));
        alert("Failed to send command: " + (data.error || res.statusText));
      }
    } catch (err) {
      console.error("Error sending command:", err);
      alert("Failed to send command. See console for details.");
    }
  };

  const downloadImplant = async (uniqueToken) => {
    const token = localStorage.getItem("token");
    try {
      const res = await fetch(
        `${API_BASE}/implants/${uniqueToken}/download`,
        {
          headers: { Authorization: `Bearer ${token}` },
        }
      );
      if (!res.ok) {
          const errorData = await res.json().catch(() => null);
          if (errorData && errorData.error) {
            throw new Error(errorData.error);
          }
          throw new Error(`Server responded with ${res.status}: ${res.statusText}`);
      }
      const blob = await res.blob();
      
      let filename = `implant_${uniqueToken}`; 
      const contentDisposition = res.headers.get('content-disposition');
      if (contentDisposition) {
          const filenameMatch = contentDisposition.match(/filename="?(.+?)"?(;|$)/i);
          if (filenameMatch && filenameMatch[1]) {
              filename = filenameMatch[1];
          }
      }
      
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
    } catch (err) {
      console.error("Download error:", err);
      alert("Failed to download implant: " + err.message);
    }
  };

  const handleOpenTerminal = (uniqueToken) => {
    setSelectedImplant(uniqueToken);
  };

  const handleCloseTerminal = () => {
    setSelectedImplant(null);
  };
  
  const getStatusColor = (status) => {
    switch (status?.toLowerCase()) {
      case 'online': return 'bg-green-100 text-green-800';
      case 'offline': return 'bg-red-100 text-red-800';
      case 'new': return 'bg-blue-100 text-blue-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };


  return (
    <div className="container mx-auto p-4 md:p-6 bg-gray-100 min-h-screen">
      <div className="bg-white shadow-xl rounded-lg p-6">
        <h2 className="text-2xl md:text-3xl font-bold mb-6 text-gray-800">Implant Management Dashboard</h2>

        <button
          onClick={generateImplant}
          className="bg-indigo-600 text-white px-5 py-2.5 rounded-lg mb-6 hover:bg-indigo-700 transition-colors shadow-md focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-opacity-50"
        >
          Generate & Download Implant
        </button>

        {implants.length === 0 && (
            <p className="text-gray-500 text-center py-10">No implants found. Generate one to get started!</p>
        )}

        {implants.length > 0 && (
          <div className="overflow-x-auto shadow-md rounded-lg">
            <table className="w-full border-collapse text-left">
              <thead className="bg-gray-200">
                <tr>
                  <th className="p-3 text-sm font-semibold text-gray-700 uppercase tracking-wider">ID</th>
                  <th className="p-3 text-sm font-semibold text-gray-700 uppercase tracking-wider">Unique Token</th>
                  <th className="p-3 text-sm font-semibold text-gray-700 uppercase tracking-wider">Status</th>
                  <th className="p-3 text-sm font-semibold text-gray-700 uppercase tracking-wider">Last Seen</th>
                  <th className="p-3 text-sm font-semibold text-gray-700 uppercase tracking-wider">IP Address</th>
                  <th className="p-3 text-sm font-semibold text-gray-700 uppercase tracking-wider text-center">Actions</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {implants.map((implant) => (
                  <tr key={implant.id} className="hover:bg-gray-50 transition-colors">
                    <td className="p-3 text-sm text-gray-700 whitespace-nowrap">{implant.id}</td>
                    <td className="p-3 text-sm text-gray-700 font-mono whitespace-nowrap">{implant.unique_token}</td>
                    <td className="p-3 text-sm text-gray-700 whitespace-nowrap">
                      <span className={`px-2.5 py-0.5 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        getStatusColor(implant.status)
                      }`}>
                        {implant.status || 'N/A'}
                      </span>
                    </td>
                    <td className="p-3 text-sm text-gray-700 whitespace-nowrap">
                      {implant.last_seen ? new Date(implant.last_seen).toLocaleString() : 'Never'}
                    </td>
                    <td className="p-3 text-sm text-gray-700 whitespace-nowrap">{implant.ip_address || 'N/A'}</td>
                    <td className="p-3 text-sm text-gray-700 whitespace-nowrap text-center space-x-1">
                      <button
                        className="bg-sky-500 text-white px-3 py-1.5 rounded-md hover:bg-sky-600 transition-colors text-xs font-medium"
                        onClick={() => handleOpenTerminal(implant.unique_token)}
                        title="Open interactive terminal"
                      >
                        Terminal
                      </button>
                      <button
                        className="bg-red-600 text-white px-3 py-1.5 rounded-md hover:bg-red-700 transition-colors text-xs font-medium"
                        onClick={() => deleteImplant(implant.unique_token)}
                        title="Delete implant"
                      >
                        Delete
                      </button>
                      <button
                        className="bg-green-500 text-white px-3 py-1.5 rounded-md hover:bg-green-600 transition-colors text-xs font-medium"
                        onClick={() => sendQuickCommand(implant.unique_token)}
                        title="Send a quick one-off command"
                      >
                        Quick Cmd
                      </button>
                      <button
                        className="bg-yellow-500 text-white px-3 py-1.5 rounded-md hover:bg-yellow-600 transition-colors text-xs font-medium"
                        onClick={() => downloadImplant(implant.unique_token)}
                        title="Download implant binary"
                      >
                        Download
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {selectedImplant && (
        <div 
          className="fixed inset-0 bg-black bg-opacity-75 flex items-center justify-center p-4 z-[100] backdrop-blur-sm"
          onClick={(e) => { if (e.target === e.currentTarget) handleCloseTerminal(); }} 
        >
          <Terminal
            implantID={selectedImplant}
            onClose={handleCloseTerminal}
          />
        </div>
      )}
    </div>
  );
}

export default Dashboard;