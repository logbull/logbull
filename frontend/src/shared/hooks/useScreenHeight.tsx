import { useEffect, useState } from 'react';

/**
 * This hook detects the full screen height
 * and adjusts dynamically, particularly for iOS where
 * 100vh or 100dvh can behave unexpectedly when the keyboard opens.
 *
 * It uses visualViewport for better handling on iOS devices.
 *
 * @returns screenHeight
 */
export function useScreenHeight(): number {
  const [screenHeight, setScreenHeight] = useState<number>(900);

  useEffect(() => {
    const updateHeight = () => {
      const height = window.visualViewport ? window.visualViewport.height : window.innerHeight;
      setScreenHeight(height);
    };

    updateHeight(); // Set initial height
    window.addEventListener('resize', updateHeight);

    // For devices with visualViewport (like iOS), also listen to viewport changes
    if (window.visualViewport) {
      window.visualViewport.addEventListener('resize', updateHeight);
    }

    return () => {
      window.removeEventListener('resize', updateHeight);
      if (window.visualViewport) {
        window.visualViewport.removeEventListener('resize', updateHeight);
      }
    };
  }, []);

  return screenHeight;
}
