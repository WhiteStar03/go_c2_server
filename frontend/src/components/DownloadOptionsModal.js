import React, { useState, useEffect } from 'react';

function DownloadOptionsModal({ isOpen, onClose, onConfirm, implantToken, targetOS, defaultC2IP }) {
  const [c2IP, setC2IP] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    if (isOpen) {
      // Pre-fill with defaultC2IP if provided, otherwise, it will be empty.
      setC2IP(defaultC2IP || '');
      setError('');
    }
  }, [isOpen, implantToken, defaultC2IP]); // Ensured defaultC2IP is in dependency array

  if (!isOpen) return null;

  const handleSubmit = () => {
    const trimmedC2IP = c2IP.trim();
    if (!trimmedC2IP) {
      setError('C2 IP Address / Hostname cannot be empty for download.');
      return;
    }

    const ipPattern = /^(?!0)(?!.*\.$)((1?\d?\d|25[0-5]|2[0-4]\d)(\.|$)){4}$/;
    const hostAndPortPattern = /^([a-zA-Z0-9.-]+|\[[0-9a-fA-F:]+\]):([0-9]{1,5})$/;
    let isValid = false;

    if (trimmedC2IP === "localhost" || trimmedC2IP.startsWith("localhost:")) {
        isValid = true;
    } else if (hostAndPortPattern.test(trimmedC2IP)) {
        isValid = true;
    } else if (trimmedC2IP.includes(':')) {
         setError('Invalid C2 IP Address or Hostname format with port. Examples: 192.168.1.100:8080, yourdomain.com:443, [::1]:80');
         return;
    } else {
        if (ipPattern.test(trimmedC2IP) || /^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/.test(trimmedC2IP)) {
            isValid = true;
        }
    }

    if (!isValid) {
        setError('Invalid C2 IP Address or Hostname format. Examples: 192.168.1.100, yourdomain.com, 10.0.0.5:8080');
        return;
    }

    setError('');
    onConfirm(implantToken, targetOS, trimmedC2IP);
  };

  const getOSDisplayName = (os) => {
    if (!os) return 'N/A';
    return os.charAt(0).toUpperCase() + os.slice(1);
  }

  return (
    <div
      className="fixed inset-0 bg-black bg-opacity-60 backdrop-blur-sm flex items-center justify-center p-4 z-[170]"
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div className="bg-white rounded-lg shadow-xl p-6 w-full max-w-md transform transition-all">
        <div className="flex items-center justify-between mb-5">
          <h3 className="text-xl font-semibold text-gray-800">Configure & Download Implant</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-6 h-6">
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-md">
            <p className="text-sm text-blue-700">
                Implant Token: <strong className="font-mono">{implantToken}</strong>
            </p>
            <p className="text-sm text-blue-700">
                Target OS: <strong className="font-medium">{getOSDisplayName(targetOS)}</strong> (pre-configured)
            </p>
        </div>


        <div className="mb-1">
          <label htmlFor="c2_ip" className="block text-sm font-medium text-gray-700 mb-1">
            C2 Server IP Address / Hostname <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            id="c2_ip"
            name="c2_ip"
            value={c2IP}
            onChange={(e) => {
              setC2IP(e.target.value);
              if (error) setError('');
            }}
            placeholder="e.g., 192.168.0.110 or yourdomain.com:8080"
            className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 sm:text-sm text-gray-900 ${ // <-- Added text-gray-900 HERE
              error ? 'border-red-500 focus:ring-red-500 focus:border-red-500' : 'border-gray-300 focus:ring-indigo-500 focus:border-indigo-500'
            }`}
          />
           <p className="mt-1 text-xs text-gray-500">
            Enter the IP address or hostname of your C2 server. You can include a port (e.g., 192.168.0.110:8080).
          </p>
          {error && <p className="mt-1 text-xs text-red-600">{error}</p>}
        </div>


        <div className="mt-6 flex justify-end space-x-3">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-gray-300"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-opacity-50"
          >
            Download
          </button>
        </div>
      </div>
    </div>
  );
}

export default DownloadOptionsModal;