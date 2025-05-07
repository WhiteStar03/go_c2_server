import React, { useEffect, useState, useRef } from "react";

const API_BASE = "http://localhost:8080/api";

// A simple X icon for the close button
const CloseIcon = () => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    fill="none"
    viewBox="0 0 24 24"
    strokeWidth={1.5}
    stroke="currentColor"
    className="w-5 h-5"
  >
    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
  </svg>
);

export default function Terminal({ implantID, onClose }) {
  const [logs, setLogs] = useState([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const polling = useRef(true); // To control polling loop
  const containerRef = useRef(null); // For scrolling

  // Scroll to bottom on new logs
  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [logs]);

  // Poll for new commands and their outputs
  useEffect(() => {
    polling.current = true; // Ensure polling is active when component mounts/implantID changes

    async function fetchLogs() {
      if (!polling.current) return; // Stop if polling is disabled

      try {
        const token = localStorage.getItem("token");
        const res = await fetch(
          `${API_BASE}/implants/${implantID}/commands`,
          { headers: { Authorization: `Bearer ${token}` } }
        );

        if (!res.ok) {
          if (res.status === 404) {
            console.warn(`Implant ${implantID} not found while fetching logs. It might have been deleted.`);
            // Optionally, trigger onClose if implant is gone for a long time or specific error
          } else {
            console.error(`Error fetching logs: ${res.status} ${res.statusText}`);
          }
          return;
        }

        const { commands } = await res.json();
        
        setLogs(prevLogs => {
            const serverCommands = commands.map(cmd => ({
                id: cmd.id,
                command: cmd.command,
                output: cmd.output || (cmd.status === 'pending' ? "…waiting for output…" : "<no output yet>"),
                status: cmd.status,
            }));

            // Keep optimistic logs that are in an error state and whose command isn't represented in the new server data
            const persistentOptimisticErrors = prevLogs.filter(log =>
                typeof log.id === 'string' && log.id.startsWith('optimistic-') && log.status === 'error' &&
                !serverCommands.some(sc => sc.command === log.command) 
            );
            
            const combinedLogs = [...serverCommands, ...persistentOptimisticErrors];
            
            // Sort logs: server (numeric ID) logs first by ID, then optimistic (string ID) logs
            return combinedLogs.sort((a, b) => {
                const aIsNum = typeof a.id === 'number';
                const bIsNum = typeof b.id === 'number';
                if (aIsNum && bIsNum) return a.id - b.id;
                if (aIsNum) return -1; 
                if (bIsNum) return 1;  
                return String(a.id).localeCompare(String(b.id)); 
            });
        });

      } catch (err) {
        console.error("Terminal fetch error:", err);
      }
    }

    fetchLogs(); // Initial load
    const intervalId = setInterval(fetchLogs, 3000); // Poll every 3 seconds

    return () => {
      polling.current = false; // Stop polling on component unmount
      clearInterval(intervalId);
    };
  }, [implantID]); // Re-run if implantID changes

  // Send a new command
  const sendCommand = async () => {
    if (!input.trim()) return;
    setLoading(true);

    const optimisticId = `optimistic-${Date.now()}`;
    const commandToSend = input;

    // Optimistically add command to logs
    setLogs(prev => [
      ...prev,
      { id: optimisticId, command: commandToSend, output: "…sending command…", status: "pending" },
    ]);
    setInput(""); // Clear input immediately

    try {
      const token = localStorage.getItem("token");
      const res = await fetch(`${API_BASE}/send-command`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          implant_id: implantID,
          command: commandToSend,
        }),
      });

      if (!res.ok) {
        const errorData = await res.json().catch(() => ({ error: "Failed to send command" }));
        const errorMessage = `Error: ${errorData.error || res.statusText}`;
        // Update the optimistic log entry with the error
        setLogs(prevLogs => prevLogs.map(log => 
            log.id === optimisticId 
            ? {...log, output: errorMessage, status: "error"} 
            : log
        ));
      }
      // If successful, poller will pick up the actual command result.
    } catch (err) {
      const errorMessage = `Error: ${err.message || "Network error"}`;
      setLogs(prevLogs => prevLogs.map(log => 
          log.id === optimisticId 
          ? {...log, output: errorMessage, status: "error"}
          : log
      ));
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    polling.current = false; // Explicitly stop polling before closing
    onClose();
  };
  
  // Prepare logs for display with appropriate styling for status/output
  const displayLogs = logs.map(log => {
    let outputDisplay = log.output;
    if (log.status === 'pending' && (log.output === '…waiting for output…' || log.output === '…sending command…')) {
      outputDisplay = <span className="text-yellow-400 italic">{log.output}</span>;
    } else if (log.status === 'error') {
      outputDisplay = <span className="text-red-400">{log.output}</span>;
    } else if (log.output === "<no output yet>" || log.output === "<no output>") {
      outputDisplay = <span className="text-gray-500 italic">{log.output}</span>;
    }
    return { ...log, outputDisplay };
  });

  return (
    <div className="flex flex-col bg-gray-900 text-gray-100 rounded-xl shadow-2xl w-full max-w-3xl mx-auto h-[70vh] overflow-hidden">
      {/* Header */}
      <div className="flex justify-between items-center bg-gray-800 px-4 py-3 border-b border-gray-700">
        <span className="font-semibold text-gray-300">
          Shell Access: {implantID}
        </span>
        <button
          onClick={handleClose}
          className="text-gray-500 hover:text-red-500 transition-colors"
          aria-label="Close terminal"
        >
          <CloseIcon />
        </button>
      </div>

      {/* Log output */}
      <div
        ref={containerRef}
        className="flex-grow bg-black/80 text-sm p-4 overflow-y-auto space-y-3 font-mono scrollbar-thin scrollbar-thumb-gray-700 scrollbar-track-gray-800"
      >
        {displayLogs.length === 0 ? (
          <div className="text-gray-500 italic">No commands executed yet. Type a command below and press Enter.</div>
        ) : (
          displayLogs.map((log) => (
            <div key={log.id}>
              <div className="flex items-baseline">
                <span className="text-sky-400 mr-2 select-none">❯</span>
                <pre className="whitespace-pre-wrap break-all text-gray-100">{log.command}</pre>
              </div>
              {(log.output || log.status === 'pending' || log.status === 'error') && ( // Show output area if there's content or it's an active state
                <pre className={`whitespace-pre-wrap break-all mt-1 pl-[calc(1ch + 0.5rem)] ${ // Dynamic padding based on prompt
                    log.status === 'error' ? 'text-red-400' : 'text-gray-400' 
                }`}>
                    {log.outputDisplay}
                </pre>
              )}
            </div>
          ))
        )}
      </div>

      {/* Input bar */}
      <form onSubmit={(e) => { e.preventDefault(); sendCommand(); }} 
            className="flex items-center bg-gray-800 p-3 border-t border-gray-700">
        <span className="text-sky-400 mr-2 font-mono text-sm select-none">❯</span>
        <input
          type="text"
          className="flex-grow bg-transparent text-gray-100 placeholder-gray-500 focus:outline-none font-mono text-sm"
          placeholder="Enter command (e.g., whoami, ls /tmp)"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              sendCommand();
            }
          }}
          disabled={loading}
          autoFocus
        />
        <button
          type="submit"
          disabled={loading || !input.trim()}
          className="bg-sky-600 hover:bg-sky-700 text-white px-4 py-1.5 rounded-md text-sm font-semibold transition-colors disabled:opacity-50 disabled:cursor-not-allowed ml-3"
        >
          {loading ? "Sending…" : "Send"}
        </button>
      </form>
    </div>
  );
}