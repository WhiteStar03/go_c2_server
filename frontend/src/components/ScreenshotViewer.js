import React, { useState, useEffect, useRef } from 'react';

// Icons (assuming these are defined as in your original code)
const CloseIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-6 h-6">
    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
  </svg>
);
const ChevronLeftIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-8 h-8">
    <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" />
  </svg>
);
const ChevronRightIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-8 h-8">
    <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
  </svg>
);
const PlayIcon = () => (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-6 h-6">
        <path strokeLinecap="round" strokeLinejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 010 1.971l-11.54 6.347a1.125 1.125 0 01-1.667-.985V5.653z" />
    </svg>
);
const PauseIcon = () => (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-6 h-6">
        <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 5.25v13.5m-7.5-13.5v13.5" />
    </svg>
);

// --- NEW FULLSCREEN ICONS ---
const FullScreenEnterIcon = () => (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-5 h-5 sm:w-6 sm:h-6">
        <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9.75 9.75M20.25 3.75v4.5m0-4.5h-4.5m4.5 0L14.25 9.75M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9.75 14.25m10.5 6L14.25 14.25" />
    </svg>
);

const FullScreenExitIcon = () => (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-5 h-5 sm:w-6 sm:h-6">
        <path strokeLinecap="round" strokeLinejoin="round" d="M9 9V4.5M9 9H4.5M9 9L3.75 3.75M9 15v4.5M9 15H4.5M9 15l-5.25 5.25M15 9V4.5M15 9h4.5M15 9l5.25-5.25M15 15v4.5M15 15h4.5M15 15l5.25 5.25" />
    </svg>
);


const C2_IMAGE_BASE_URL = 'http://localhost:8080';

function ScreenshotViewer({
  isOpen,
  onClose,
  implantId,
  screenshots: screenshotPathsFromProps,
  initialScreenshotPath,
  mode = 'gallery',
  onStreamStopRequested,
}) {
  const [currentImageSrc, setCurrentImageSrc] = useState('');
  const [galleryImages, setGalleryImages] = useState([]);
  const [currentGalleryIndex, setCurrentGalleryIndex] = useState(0);
  const [isLoadingInitial, setIsLoadingInitial] = useState(true);
  const [errorMessage, setErrorMessage] = useState('');
  const [isPlayingSlideshow, setIsPlayingSlideshow] = useState(false);
  const slideshowIntervalRef = useRef(null);
  const imageRef = useRef(null);
  const viewerContentRef = useRef(null); // Ref for the element to make fullscreen
  const [isFullScreen, setIsFullScreen] = useState(false);


  // Effect for handling viewer open/close and initial data setup
  useEffect(() => {
    // ... (this useEffect remains largely the same)
    if (isOpen) {
      setIsLoadingInitial(true);
      setErrorMessage('');
      setIsPlayingSlideshow(false);
      if (slideshowIntervalRef.current) clearInterval(slideshowIntervalRef.current);
      setIsFullScreen(false); // Reset fullscreen state on open

      const validPaths = (screenshotPathsFromProps || [])
        .filter(s => typeof s === 'string' && s.startsWith('c2_screenshots/'));

      if (mode === 'livestream') {
        setGalleryImages([]);
        if (validPaths.length > 0) {
          setCurrentImageSrc(`${C2_IMAGE_BASE_URL}/${validPaths[0]}`);
        } else {
          setCurrentImageSrc('');
          setErrorMessage("Waiting for livestream frames...");
        }
      } else { 
        setGalleryImages(validPaths);
        if (validPaths.length > 0) {
          let initialIdx = 0;
          if (initialScreenshotPath) {
            const foundIdx = validPaths.indexOf(initialScreenshotPath);
            if (foundIdx !== -1) initialIdx = foundIdx;
          }
          setCurrentGalleryIndex(initialIdx);
          setCurrentImageSrc(`${C2_IMAGE_BASE_URL}/${validPaths[initialIdx]}`);
        } else {
          setCurrentImageSrc('');
          setErrorMessage("No screenshots available for this implant.");
        }
      }
      setIsLoadingInitial(false);
    } else {
      if (slideshowIntervalRef.current) {
        clearInterval(slideshowIntervalRef.current);
        slideshowIntervalRef.current = null;
      }
      setIsPlayingSlideshow(false);
      if (mode === 'livestream' && implantId && onStreamStopRequested) {
        onStreamStopRequested(implantId);
      }
      // Exit fullscreen if viewer is closed while in fullscreen
      if (isFullScreen && document.fullscreenElement) {
        document.exitFullscreen().catch(err => console.error("Error exiting fullscreen on close:", err));
      }
      setIsFullScreen(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, implantId, mode, initialScreenshotPath]);

  // Effect to handle updates to screenshotPathsFromProps (polling)
  useEffect(() => {
    // ... (this useEffect remains largely the same)
    if (!isOpen || isLoadingInitial) return;

    const validPaths = (screenshotPathsFromProps || [])
      .filter(s => typeof s === 'string' && s.startsWith('c2_screenshots/'));

    if (mode === 'livestream') {
      if (validPaths.length > 0) {
        const newSrc = `${C2_IMAGE_BASE_URL}/${validPaths[0]}`;
        if (newSrc !== currentImageSrc) {
          setCurrentImageSrc(newSrc);
        }
        if (errorMessage === "Waiting for livestream frames...") setErrorMessage('');
      } else {
        if (currentImageSrc) setCurrentImageSrc(''); 
        if (!errorMessage) setErrorMessage("Waiting for livestream frames...");
      }
    } else { 
      setGalleryImages(validPaths);
      if (validPaths.length === 0) {
        if (!errorMessage) setErrorMessage("No screenshots available for this implant.");
        if (currentImageSrc) setCurrentImageSrc('');
      } else {
        let newIndex = currentGalleryIndex;
        const currentPath = currentImageSrc.replace(`${C2_IMAGE_BASE_URL}/`, '');
        const existingIdx = validPaths.indexOf(currentPath);

        if (existingIdx !== -1) {
            newIndex = existingIdx;
        } else if (newIndex >= validPaths.length) {
            newIndex = 0; 
        }
        setCurrentGalleryIndex(newIndex);
        
        const newSrc = `${C2_IMAGE_BASE_URL}/${validPaths[newIndex]}`;
        if (newSrc !== currentImageSrc) {
            setCurrentImageSrc(newSrc);
        }
        if (errorMessage) setErrorMessage('');
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [screenshotPathsFromProps, isOpen, mode, isLoadingInitial]);


  // Effect for gallery navigation changing the image
  useEffect(() => {
    // ... (this useEffect remains largely the same)
    if (isOpen && mode === 'gallery' && galleryImages.length > 0 && galleryImages[currentGalleryIndex]) {
        const newSrc = `${C2_IMAGE_BASE_URL}/${galleryImages[currentGalleryIndex]}`;
        if (newSrc !== currentImageSrc) {
          setCurrentImageSrc(newSrc);
          setErrorMessage(''); 
        }
      }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentGalleryIndex, isOpen, mode]); // galleryImages change will also trigger if needed


  // Effect for handling fullscreen changes (e.g., user pressing ESC)
  useEffect(() => {
    const handleFullScreenChange = () => {
      setIsFullScreen(!!document.fullscreenElement);
    };

    document.addEventListener('fullscreenchange', handleFullScreenChange);
    return () => {
      document.removeEventListener('fullscreenchange', handleFullScreenChange);
    };
  }, []);


  // Cleanup on unmount
  useEffect(() => {
    // ... (this useEffect remains largely the same)
    return () => {
        if (slideshowIntervalRef.current) clearInterval(slideshowIntervalRef.current);
        if (isOpen && mode === 'livestream' && implantId && onStreamStopRequested) {
          onStreamStopRequested(implantId);
        }
        // Ensure exiting fullscreen if component unmounts while in fullscreen
        if (document.fullscreenElement) {
          document.exitFullscreen().catch(err => console.error("Error exiting fullscreen on unmount:", err));
        }
      };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, mode, implantId]);

  const handleNextGallery = () => { /* ... (same) ... */ 
    if (galleryImages.length === 0) return;
    setCurrentGalleryIndex((prevIndex) => (prevIndex + 1) % galleryImages.length);
  };
  const handlePrevGallery = () => { /* ... (same) ... */ 
    if (galleryImages.length === 0) return;
    setCurrentGalleryIndex((prevIndex) => (prevIndex - 1 + galleryImages.length) % galleryImages.length);
  };
  const toggleSlideshow = () => { /* ... (same) ... */ 
    if (galleryImages.length <= 1) return;
    if (isPlayingSlideshow) {
      clearInterval(slideshowIntervalRef.current);
      slideshowIntervalRef.current = null;
    } else {
      handleNextGallery(); 
      slideshowIntervalRef.current = setInterval(() => {
        handleNextGallery();
      }, 2000); 
    }
    setIsPlayingSlideshow(!isPlayingSlideshow);
  };
  const handleImageError = (e, srcPath) => { /* ... (same) ... */ 
    console.error("Error loading image:", srcPath, e);
    setErrorMessage(`Failed to load image: ${srcPath.split('/').pop()}`);
    setCurrentImageSrc('');
  };

  const toggleFullScreen = () => {
    if (!viewerContentRef.current) return;

    if (!document.fullscreenElement) {
      viewerContentRef.current.requestFullscreen()
        .then(() => setIsFullScreen(true))
        .catch(err => {
          alert(`Error attempting to enable full-screen mode: ${err.message} (${err.name})`);
          console.error(`Error attempting to enable full-screen mode: ${err.message} (${err.name})`);
        });
    } else {
      document.exitFullscreen()
        .then(() => setIsFullScreen(false))
        .catch(err => console.error(`Error attempting to exit full-screen mode: ${err.message} (${err.name})`));
    }
  };


  if (!isOpen) return null;

  return (
    <div // This is the backdrop
      className="fixed inset-0 bg-black bg-opacity-80 backdrop-blur-md flex flex-col items-center justify-center p-3 sm:p-4 z-[200]"
      onClick={(e) => { if (e.target === e.currentTarget && !isFullScreen) onClose(); }} // Only close by backdrop click if not fullscreen
    >
      {/* This ref'd div is what will go fullscreen */}
      <div ref={viewerContentRef} className={`relative bg-gray-900 p-3 sm:p-4 rounded-lg shadow-2xl w-full flex flex-col 
        ${isFullScreen ? 'max-w-full max-h-full h-screen' : 'max-w-3xl max-h-[90vh]'}`}>
        
        {/* Header */}
        <div className="flex justify-between items-center mb-3 pb-3 border-b border-gray-700">
          <h3 className="text-lg sm:text-xl font-semibold text-gray-100 truncate">
            {mode === 'livestream' ? 'Livestream' : 'Screenshots'} - <span className="font-mono text-xs sm:text-sm opacity-80">{implantId || "N/A"}</span>
          </h3>
          <div className="flex items-center space-x-2 sm:space-x-3">
            <button 
                onClick={toggleFullScreen} 
                className="text-gray-400 hover:text-white transition-colors p-1"
                title={isFullScreen ? "Exit Fullscreen" : "Enter Fullscreen"}
            >
                {isFullScreen ? <FullScreenExitIcon /> : <FullScreenEnterIcon />}
            </button>
            {!isFullScreen && ( // Only show close button if not in fullscreen (ESC handles fullscreen exit)
                <button onClick={onClose} className="text-gray-400 hover:text-red-500 transition-colors p-1">
                    <CloseIcon />
                </button>
            )}
          </div>
        </div>

        {/* Image Display Area */}
        <div className={`flex-grow flex items-center justify-center overflow-hidden relative bg-gray-800 rounded
            ${isFullScreen ? 'h-full' : 'min-h-[250px] sm:min-h-[300px]'}`}>
          {isLoadingInitial && <p className="text-gray-400 p-5">Loading...</p>}
          
          {!isLoadingInitial && errorMessage && (
            <div className="text-red-400 p-5 text-center">
              <p>{errorMessage}</p>
            </div>
          )}

          {!isLoadingInitial && !errorMessage && currentImageSrc && (
            <img
              ref={imageRef}
              key={currentImageSrc} 
              src={currentImageSrc}
              alt={`${mode === 'livestream' ? 'Livestream Frame' : `Screenshot ${currentGalleryIndex + 1}`} for ${implantId}`}
              className={`max-w-full object-contain rounded
                ${isFullScreen ? 'max-h-full h-full' : 'max-h-[calc(85vh-120px)] sm:max-h-[calc(85vh-150px)]'}`}
              onError={(e) => handleImageError(e, currentImageSrc)}
            />
          )}
           {!isLoadingInitial && !errorMessage && !currentImageSrc && (
             <p className="text-gray-400 p-5">{mode === 'livestream' ? "Connecting to live feed..." : "No screenshots found."}</p>
           )}
        </div>

        {/* Controls Area - Potentially hide or simplify in fullscreen */}
        {!isFullScreen && (
            <div className="mt-3 pt-3 border-t border-gray-700">
                {mode === 'gallery' && !isLoadingInitial && galleryImages.length > 0 && (
                <div className="flex items-center justify-between">
                    <button
                    onClick={handlePrevGallery}
                    disabled={galleryImages.length <= 1 || isPlayingSlideshow}
                    className="p-2 text-gray-300 hover:text-white disabled:text-gray-600 disabled:cursor-not-allowed transition-colors rounded-full hover:bg-gray-700"
                    title="Previous Screenshot"
                    >
                    <ChevronLeftIcon />
                    </button>

                    <div className="flex items-center space-x-3 sm:space-x-4">
                        <button
                            onClick={toggleSlideshow}
                            disabled={galleryImages.length <= 1}
                            className="p-2 text-gray-300 hover:text-white disabled:text-gray-600 disabled:cursor-not-allowed transition-colors rounded-full hover:bg-gray-700"
                            title={isPlayingSlideshow ? "Pause Slideshow" : "Play Slideshow (2s interval)"}
                        >
                            {isPlayingSlideshow ? <PauseIcon /> : <PlayIcon />}
                        </button>
                        <span className="text-xs sm:text-sm text-gray-400">
                            {galleryImages.length > 0 ? `${currentGalleryIndex + 1} / ${galleryImages.length}` : '0 / 0'}
                        </span>
                    </div>

                    <button
                    onClick={handleNextGallery}
                    disabled={galleryImages.length <= 1 || isPlayingSlideshow}
                    className="p-2 text-gray-300 hover:text-white disabled:text-gray-600 disabled:cursor-not-allowed transition-colors rounded-full hover:bg-gray-700"
                    title="Next Screenshot"
                    >
                    <ChevronRightIcon />
                    </button>
                </div>
                )}
                {mode === 'livestream' && !isLoadingInitial && (
                    <div className="flex items-center justify-center">
                        <span className={`text-xs sm:text-sm flex items-center ${currentImageSrc && !errorMessage ? 'text-green-400' : 'text-yellow-400'}`}>
                            <span className={`w-2 h-2 rounded-full mr-2 ${currentImageSrc && !errorMessage ? 'bg-green-500 animate-pulse' : 'bg-yellow-500'}`}></span>
                            {currentImageSrc && !errorMessage ? "Live Feed Active" : (errorMessage || "Connecting...")}
                        </span>
                    </div>
                )}
            </div>
        )}
      </div>
    </div>
  );
}

export default ScreenshotViewer;