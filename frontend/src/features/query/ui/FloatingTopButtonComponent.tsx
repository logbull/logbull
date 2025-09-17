import { UpOutlined } from '@ant-design/icons';
import { Button } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';

interface Props {
  containerRef: React.RefObject<HTMLDivElement | null>;
  scrollThreshold?: number;
}

/**
 * FloatingTopButtonComponent - A floating scroll-to-top button that appears when scrolling
 *
 * Features:
 * - Throttled scroll detection for optimal performance
 * - Smooth scroll-to-top animation
 * - Emerald theme styling consistent with app design
 * - Appears when scrolled past threshold (default 200px)
 * - Fixed positioning with high z-index
 */

export const FloatingTopButtonComponent = ({
  containerRef,
  scrollThreshold = 200,
}: Props): React.JSX.Element | null => {
  // Scroll state and refs
  const [showScrollToTop, setShowScrollToTop] = useState(false);
  const scrollTimeoutRef = useRef<number | null>(null);

  // Scroll detection to show/hide floating button
  const handleScroll = useCallback(() => {
    if (!containerRef.current) {
      return;
    }

    const container = containerRef.current;
    const scrollTop = container.scrollTop;

    // Show button if scrolled down more than threshold
    const shouldShowButton = scrollTop > scrollThreshold;

    // Only update state if the button visibility actually changes
    setShowScrollToTop((prev) => {
      if (prev !== shouldShowButton) {
        return shouldShowButton;
      }
      return prev;
    });
  }, [containerRef, scrollThreshold]);

  // Throttled version of handleScroll to improve performance
  const throttledHandleScroll = useCallback(() => {
    if (scrollTimeoutRef.current) {
      return;
    }

    scrollTimeoutRef.current = setTimeout(() => {
      handleScroll();
      scrollTimeoutRef.current = null;
    }, 16); // ~60fps
  }, [handleScroll]);

  // Scroll to top functionality
  const scrollToTop = useCallback(() => {
    if (containerRef.current) {
      containerRef.current.scrollTo({
        top: 0,
        behavior: 'smooth',
      });
    }
  }, [containerRef]);

  // Attach scroll listener and check initial position
  useEffect(() => {
    const container = containerRef.current;
    if (!container) {
      return;
    }

    // Check initial scroll position
    handleScroll();

    container.addEventListener('scroll', throttledHandleScroll);
    return () => {
      container.removeEventListener('scroll', throttledHandleScroll);
      // Clear any pending timeout on cleanup
      if (scrollTimeoutRef.current) {
        clearTimeout(scrollTimeoutRef.current);
        scrollTimeoutRef.current = null;
      }
    };
  }, [handleScroll, throttledHandleScroll]);

  // Don't render if button shouldn't be shown
  if (!showScrollToTop) {
    return null;
  }

  return (
    <div
      className="fixed z-50"
      style={{
        top: '100px',
        right: '25px',
        zIndex: 999999,
      }}
    >
      <Button
        type="primary"
        className="border-emerald-600 bg-emerald-600 shadow-lg hover:border-emerald-700 hover:bg-emerald-700"
        shape="circle"
        icon={<UpOutlined />}
        size="large"
        onClick={scrollToTop}
      />
    </div>
  );
};
