import React, { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import Terminal from "./Terminal";

function Dashboard() {
  const [implants, setImplants] = useState([]);
  const [selectedImplant, setSelectedImplant] = useState(null);
  const navigate = useNavigate();

  useEffect(() => {
    fetchImplants();
  }, []);

  const fetchImplants = async () => {
    const token = localStorage.getItem("token");
    try {
      const response = await fetch("http://localhost:8080/api/implants", {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await response.json();
      setImplants(data.implants || []);
    } catch (err) {
      console.error("Error fetching implants:", err);
    }
  };

  const generateImplant = async () => {
    const token = localStorage.getItem("token");
    try {
      const response = await fetch("http://localhost:8080/api/generate-implant", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
      });
      const data = await response.json();
      if (response.ok) {
        alert("Implant generated successfully!");
        fetchImplants(); // Refresh the list of implants
      } else {
        alert("Failed to generate implant: " + (data.error || "Unknown error"));
      }
    } catch (err) {
      console.error("Error generating implant:", err);
      alert("Failed to generate implant. Check the console for details.");
    }
  };

  const deleteImplant = async (id) => {
    const token = localStorage.getItem("token");
    try {
      const response = await fetch(`http://localhost:8080/api/implants/${id}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      });
      if (response.ok) {
        alert("Implant deleted successfully");
        fetchImplants(); // Refresh the list of implants
      } else {
        alert("Failed to delete implant");
      }
    } catch (err) {
      console.error("Error deleting implant:", err);
    }
  };

  const sendCommand = async (id) => {
    const token = localStorage.getItem("token");
    const command = prompt("Enter command to send:");
    if (!command) return;

    try {
      const response = await fetch("http://localhost:8080/api/send-command", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ implant_id: id, command }),
      });
      if (response.ok) {
        alert("Command sent successfully");
      } else {
        alert("Failed to send command");
      }
    } catch (err) {
      console.error("Error sending command:", err);
    }
  };

  return (
    <div className="container mx-auto p-6">
      <h2 className="text-2xl font-bold mb-4">Implant Management</h2>

      {/* Generate Implant Button */}
      <button
        onClick={generateImplant}
        className="bg-blue-500 text-white px-4 py-2 rounded mb-4 hover:bg-blue-600"
      >
        Generate New Implant
      </button>

      {/* Implant Table */}
      <table className="w-full border-collapse border border-gray-200">
        <thead>
          <tr className="bg-gray-200">
            <th className="border p-2">ID</th>
            <th className="border p-2">Unique Token</th>
            <th className="border p-2">Status</th>
            <th className="border p-2">Last Seen</th>
            <th className="border p-2">Actions</th>
          </tr>
        </thead>
        <tbody>
          {implants.map((implant) => (
            <tr key={implant.id} className="border">
              <td className="border p-2">{implant.id}</td>
              <td className="border p-2">{implant.unique_token}</td>
              <td className="border p-2">{implant.status}</td>
              <td className="border p-2">{new Date(implant.last_seen).toLocaleString()}</td>
              <td className="border p-2">
                <button
                  className="bg-blue-500 text-white px-3 py-1 rounded mr-2"
                  onClick={() => setSelectedImplant(implant.unique_token)}
                >
                  Open Terminal
                </button>
                <button
                  className="bg-red-500 text-white px-3 py-1 rounded mr-2"
                  onClick={() => deleteImplant(implant.unique_token)}
                >
                  Delete
                </button>
                <button
                  className="bg-green-500 text-white px-3 py-1 rounded"
                  onClick={() => sendCommand(implant.unique_token)}
                >
                  Send Command
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {/* Terminal for Selected Implant */}
      {selectedImplant && (
        <div className="mt-6">
          <h3 className="text-xl font-bold mb-4">Terminal for Implant {selectedImplant}</h3>
          <Terminal implantID={selectedImplant} />
        </div>
      )}
    </div>
  );
}

export default Dashboard;