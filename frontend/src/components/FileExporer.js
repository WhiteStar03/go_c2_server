// src/components/FileExplorer.js
import React, { useState, useEffect, useCallback } from 'react';

const API_BASE = "/api";
// Basic Icons (Consider a library like react-icons for more)
const FolderIcon = () => <span className="mr-2">📁</span>;
const FileIcon = () => <span className="mr-2">📄</span>;
const SymlinkIcon = () => <span className="mr-2">🔗</span>;

function FileExplorer({ implantID, initialPath = ".", onClose }) {
  const [currentPath, setCurrentPath] = useState(initialPath); // Path displayed and sent to backend
  const [pathInput, setPathInput] = useState(initialPath); // For the input field
  const [items, setItems] = useState([]);
  const [fileContent, setFileContent] = useState(null); // { name: string, content: string, path: string }
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const authToken = localStorage.getItem("token");

  const fetchDirectoryListing = useCallback(async (pathToNavigate) => {
    if (!implantID) return;
    setIsLoading(true);
    setError('');
    setFileContent(null); // Clear file view when navigating

    try {
      const res = await fetch(`${API_BASE}/implants/${implantID}/fs/list`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${authToken}`,
        },
        body: JSON.stringify({ path: pathToNavigate }),
      });
      if (!res.ok) {
        const errData = await res.json().catch(() => ({ error: `Server error: ${res.status}` }));
        throw new Error(errData.error || `Failed to list directory: ${res.statusText}`);
      }
      const data = await res.json();
      setItems(data.items || []);
      setCurrentPath(data.path); // Update to the path resolved by C2
      setPathInput(data.path);   // Sync input field
    } catch (err) {
      setError(err.message);
      setItems([]);
      console.error("FS List Error:", err);
    } finally {
      setIsLoading(false);
    }
  }, [implantID, authToken]);

  useEffect(() => {
    // Fetch initial listing
    fetchDirectoryListing(initialPath);
  }, [fetchDirectoryListing, initialPath]); // Only on mount or if initialPath/fetch func changes

  const handleItemClick = (item) => {
    if (item.type === 'directory') {
      // For directories, we send its name. C2 will append it to its current CWD for the implant.
      fetchDirectoryListing(item.name);
    } else if (item.type === 'file' || item.type === 'symlink') { // Treat symlinks as files for viewing
      viewFile(item.name); // Send file name, C2 resolves full path using its CWD
    }
  };

  const viewFile = async (fileName) => {
    setIsLoading(true);
    setError('');
    // Construct the full path to the file to be viewed
    // The `currentPath` is the directory, `fileName` is the item within it.
    let separator = "/";
    // Very basic OS detection from path for separator. C2 should ideally normalize.
    if (currentPath.includes("\\") && !currentPath.includes("/")) separator = "\\";
    
    let fullFilePath;
    if (currentPath.endsWith(separator) || currentPath === "." || currentPath === "") {
        fullFilePath = currentPath === "." ? fileName : currentPath + fileName;
    } else {
        fullFilePath = currentPath + separator + fileName;
    }


    try {
      const res = await fetch(`${API_BASE}/implants/${implantID}/fs/read`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${authToken}`,
        },
        body: JSON.stringify({ path: fullFilePath }), // Send full path for reading
      });
      if (!res.ok) {
        const errData = await res.json().catch(() => ({ error: `Server error: ${res.status}` }));
        throw new Error(errData.error || `Failed to read file: ${res.statusText}`);
      }
      const data = await res.json();
      setFileContent({ name: fileName, content: data.content, path: data.path, isBinary: data.is_binary });
    } catch (err) {
      setError(err.message);
      setFileContent(null);
      console.error("FS Read Error:", err);
    } finally {
      setIsLoading(false);
    }
  };

  const handlePathSubmit = (e) => {
    e.preventDefault();
    fetchDirectoryListing(pathInput);
  };

  const navigateUp = () => {
    fetchDirectoryListing(".."); // C2 handles ".." relative to its CWD for the implant
  };

  const getIcon = (type) => {
    if (type === 'directory') return <FolderIcon />;
    if (type === 'file') return <FileIcon />;
    if (type === 'symlink') return <SymlinkIcon />;
    return <FileIcon />;
  };

  const formatBytes = (bytes, decimals = 2) => {
    if (!bytes || bytes === 0) return '0 Bytes';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
  };

  return (
    <div className="bg-gray-800 text-white p-4 rounded-lg shadow-xl h-[70vh] flex flex-col w-full max-w-4xl mx-auto">
      <div className="flex items-center mb-3">
        <h3 className="text-lg font-semibold mr-auto">File Explorer: {currentPath}</h3>
        <button onClick={onClose} className="text-gray-400 hover:text-white">×</button>
      </div>

      <form onSubmit={handlePathSubmit} className="flex mb-2">
        <button type="button" onClick={navigateUp} className="p-2 bg-gray-700 hover:bg-gray-600 rounded-l-md">Up</button>
        <input
          type="text"
          value={pathInput}
          onChange={(e) => setPathInput(e.target.value)}
          className="flex-grow p-2 bg-gray-700 border-t border-b border-gray-600 focus:border-blue-500 outline-none"
          placeholder="Enter path"
        />
        <button type="submit" className="p-2 bg-blue-600 hover:bg-blue-700 rounded-r-md">Go</button>
      </form>

      {isLoading && <div className="text-center py-4">Loading...</div>}
      {error && <div className="text-red-400 bg-red-900 p-2 rounded mb-2">{error}</div>}

      {fileContent ? (
        <div className="flex-grow flex flex-col">
          <div className="mb-2">
            <button onClick={() => setFileContent(null)} className="px-3 py-1 bg-gray-600 hover:bg-gray-500 rounded mr-2">
              ← Back to List
            </button>
            <span className="font-semibold truncate" title={fileContent.path}>{fileContent.name}</span>
          </div>
          {fileContent.isBinary ? (
             <div className="flex-grow p-3 bg-gray-900 rounded overflow-auto text-xs border border-gray-700">
                Binary file content cannot be displayed directly. (Size: {formatBytes(fileContent.content?.length || 0)})
                {/* In a real app, you might offer a download link here */}
             </div>
          ) : (
            <pre className="flex-grow p-3 bg-gray-900 rounded overflow-auto text-xs whitespace-pre-wrap break-all border border-gray-700">
              {fileContent.content || "[Empty File]"}
            </pre>
          )}
        </div>
      ) : (
        <div className="flex-grow overflow-auto border border-gray-700 rounded">
          <table className="w-full text-sm">
            <thead className="bg-gray-700 sticky top-0">
              <tr>
                <th className="p-2 text-left">Name</th>
                <th className="p-2 text-left hidden md:table-cell">Permissions</th>
                <th className="p-2 text-right hidden sm:table-cell">Size</th>
                <th className="p-2 text-left hidden md:table-cell">Modified</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item, idx) => (
                <tr key={item.name + idx} /* Ideally use a more unique key if available */
                    className="hover:bg-gray-700 cursor-pointer border-b border-gray-700 last:border-b-0"
                    onClick={() => handleItemClick(item)}
                    title={item.type === 'symlink' && item.target ? `Symlink to: ${item.target}` : item.name}
                >
                  <td className="p-2 flex items-center whitespace-nowrap">
                    {getIcon(item.type)}
                    {item.name}
                  </td>
                  <td className="p-2 font-mono hidden md:table-cell">{item.permissions || 'N/A'}</td>
                  <td className="p-2 text-right hidden sm:table-cell">
                    {item.type !== 'directory' ? formatBytes(item.size) : '-'}
                  </td>
                  <td className="p-2 hidden md:table-cell whitespace-nowrap">
                    {item.modified ? new Date(item.modified).toLocaleString() : 'N/A'}
                  </td>
                </tr>
              ))}
              {items.length === 0 && !isLoading && (
                <tr><td colSpan="4" className="text-center p-4 text-gray-500">Directory is empty or inaccessible.</td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

export default FileExplorer;