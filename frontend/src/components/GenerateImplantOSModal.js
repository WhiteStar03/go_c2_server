import React from 'react'; // Don't forget React import

function GenerateImplantOSModal({ isOpen, onClose, onConfirm, selectedOS, setSelectedOS }) {
    if (!isOpen) return null;

  const handleSubmit = () => {
    if (selectedOS) {
      onConfirm(selectedOS);
    }
  };

  return (
    <div
      className="fixed inset-0 bg-black bg-opacity-60 backdrop-blur-sm flex items-center justify-center p-4 z-[160]"
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div className="bg-white rounded-lg shadow-xl p-6 w-full max-w-sm">
        <div className="flex items-center justify-between mb-5">
          <h3 className="text-xl font-semibold text-gray-800">Generate New Implant</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-6 h-6">
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <p className="text-gray-600 mb-4">Select the target operating system for the new implant:</p>
        <div className="space-y-3 mb-6">
          <div>
            <label htmlFor="os_windows_gen" className="flex items-center p-3 rounded-md border border-gray-300 hover:bg-gray-50 cursor-pointer transition-colors has-[:checked]:bg-indigo-50 has-[:checked]:border-indigo-500">
              <input
                type="radio"
                id="os_windows_gen"
                name="target_os_generation"
                value="windows"
                checked={selectedOS === "windows"}
                onChange={(e) => setSelectedOS(e.target.value)}
                className="form-radio h-5 w-5 text-indigo-600 border-gray-300 focus:ring-indigo-500 mr-3"
              />
              <span className="text-gray-700 font-medium">Windows</span>
            </label>
          </div>
          <div>
            <label htmlFor="os_linux_gen" className="flex items-center p-3 rounded-md border border-gray-300 hover:bg-gray-50 cursor-pointer transition-colors has-[:checked]:bg-indigo-50 has-[:checked]:border-indigo-500">
              <input
                type="radio"
                id="os_linux_gen"
                name="target_os_generation"
                value="linux"
                checked={selectedOS === "linux"}
                onChange={(e) => setSelectedOS(e.target.value)}
                className="form-radio h-5 w-5 text-indigo-600 border-gray-300 focus:ring-indigo-500 mr-3"
              />
              <span className="text-gray-700 font-medium">Linux</span>
            </label>
          </div>
        </div>
        <div className="flex justify-end space-x-3">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-gray-300"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-opacity-50"
            disabled={!selectedOS}
          >
            Generate
          </button>
        </div>
      </div>
    </div>
  );
}

export default GenerateImplantOSModal; // Export it