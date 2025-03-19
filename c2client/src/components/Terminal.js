// src/components/Terminal.js
import React, { useState, useEffect } from "react";

function Terminal({ implantID }) {
  const [commands, setCommands] = useState([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const interval = setInterval(() => {
      fetchCommands();
    }, 1000); // Fetch every 3 seconds
    return () => clearInterval(interval);
  }, [implantID]);

  const fetchCommands = async () => {
    const token = localStorage.getItem("token");
    try {
      const response = await fetch(`http://localhost:8080/implants/${implantID}/commands`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await response.json();
      setCommands(data.commands || []);
    } catch (err) {
      console.error("Error fetching commands:", err);
    }
  };

  const sendCommand = async () => {
    if (!input.trim()) return;
    setLoading(true);
    const token = localStorage.getItem("token");
    try {
      const response = await fetch("http://localhost:8080/api/send-command", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ implant_id: implantID, command: input }),
      });
      const data = await response.json();
      if (response.ok) {
        setCommands((prevCommands) => [...prevCommands, { command: input, output: "Waiting for response...", status: "pending" }]);
        setInput("");
        fetchCommands(); // Fetch immediately to update
      }
    } catch (err) {
      console.error("Error sending command:", err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-black text-white p-4 rounded-lg font-mono">
      <div className="overflow-y-auto h-64 mb-4">
        {commands.map((cmd, index) => (
          <div key={index}>
            <div className="text-green-400">$ {cmd.command}</div>
            <div className={`text-white ${cmd.status === "pending" ? "text-yellow-400" : "text-white"}`}>{cmd.output}</div>
          </div>
        ))}
      </div>
      <div className="flex">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyPress={(e) => e.key === "Enter" && sendCommand()}
          className="flex-grow bg-black text-white border-none outline-none"
          placeholder="Enter command..."
          disabled={loading}
        />
        <button
          onClick={sendCommand}
          className="bg-green-500 text-white px-4 py-2 rounded ml-2"
          disabled={loading}
        >
          {loading ? "Sending..." : "Send"}
        </button>
      </div>
    </div>
  );
}

export default Terminal;
