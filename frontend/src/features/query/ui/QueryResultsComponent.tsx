import { LoadingOutlined } from '@ant-design/icons';
import { Spin } from 'antd';
import dayjs from 'dayjs';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import { type LogItem } from '../../../entity/query';

const STORAGE_KEY = 'logbull-message-length';

/**
 * Get default message length based on screen width
 */
const getDefaultMessageLength = (): number => {
  if (typeof window === 'undefined') {
    return 135; // Default for SSR
  }

  const screenWidth = window.innerWidth;

  if (screenWidth <= 1440) {
    return 135;
  } else if (screenWidth <= 1920) {
    return 100;
  } else {
    // 2K and above
    return 145;
  }
};

/**
 * Get stored message length from localStorage, fallback to screen-based default
 */
const getStoredMessageLength = (): number => {
  if (typeof window === 'undefined') {
    return getDefaultMessageLength();
  }

  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored !== null) {
      const parsed = parseInt(stored, 10);
      if (!isNaN(parsed) && parsed >= 10 && parsed <= 1000) {
        return parsed;
      }
    }
  } catch (error) {
    console.warn('Failed to read message length from localStorage:', error);
  }

  return getDefaultMessageLength();
};

interface Props {
  queryResults: LogItem[];
  totalResults: number;
  hasExecuted: boolean;
  isExecuting: boolean;
  hasMoreResults: boolean;
  onLoadMore: () => void;
}

/**
 * QueryResultsComponent - Displays log query results with infinite scroll
 *
 * Features:
 * - Results displayed in flex-based layout with fixed column widths
 * - Color-coded log levels with badges
 * - Infinite scroll loading when user scrolls to bottom
 * - Loading states during query execution
 * - Empty state when no results found
 * - Click to expand/collapse field details
 */
export const QueryResultsComponent = ({
  queryResults,
  totalResults,
  hasExecuted,
  isExecuting,
  hasMoreResults,
  onLoadMore,
}: Props): React.JSX.Element | null => {
  // States
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const [messageLength, setMessageLength] = useState<number>(getStoredMessageLength());

  // Refs
  const isLoadingMore = useRef(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Functions
  const handleScroll = useCallback(() => {
    if (isLoadingMore.current || !hasMoreResults || isExecuting) {
      return;
    }

    // Find the scrollable parent container
    const scrollContainer = containerRef.current?.closest('.overflow-y-auto') as HTMLElement;
    if (!scrollContainer) {
      return;
    }

    const scrollTop = scrollContainer.scrollTop;
    const scrollHeight = scrollContainer.scrollHeight;
    const clientHeight = scrollContainer.clientHeight;
    const scrollThreshold = 100; // Load more when 100px from bottom

    if (scrollHeight - scrollTop - clientHeight < scrollThreshold) {
      isLoadingMore.current = true;
      onLoadMore();
    }
  }, [hasMoreResults, isExecuting, onLoadMore]);

  const renderLogLevel = (level: string) => {
    const colors = {
      ERROR: 'bg-red-100 text-red-800 border-red-200',
      WARN: 'bg-yellow-100 text-yellow-800 border-yellow-200',
      INFO: 'bg-blue-100 text-blue-800 border-blue-200',
      DEBUG: 'bg-gray-100 text-gray-800 border-gray-200',
      TRACE: 'bg-purple-100 text-purple-800 border-purple-200',
      FATAL: 'bg-red-200 text-red-900 border-red-300',
      CRITICAL: 'bg-red-200 text-red-900 border-red-300',
    };

    const colorClass = colors[level as keyof typeof colors] || colors.INFO;

    return (
      <span className={`inline-block rounded border px-1 py-0.5 text-xs font-medium ${colorClass}`}>
        {level}
      </span>
    );
  };

  const truncateText = (
    text: string,
    maxLength: number,
  ): { text: string; isTruncated: boolean } => {
    if (text.length <= maxLength) {
      return { text, isTruncated: false };
    }
    return { text: text.substring(0, maxLength) + '...', isTruncated: true };
  };

  const formatFieldValue = (value: string): { formatted: string; isJson: boolean } => {
    try {
      // Try to parse as JSON
      const parsed = JSON.parse(value);
      // If successful, format with proper indentation
      return {
        formatted: JSON.stringify(parsed, null, 2),
        isJson: true,
      };
    } catch {
      // Not JSON, return as is
      return {
        formatted: value,
        isJson: false,
      };
    }
  };

  const toggleRowExpansion = (logId: string) => {
    const newExpandedRows = new Set(expandedRows);
    if (expandedRows.has(logId)) {
      newExpandedRows.delete(logId);
    } else {
      newExpandedRows.add(logId);
    }
    setExpandedRows(newExpandedRows);
  };

  const renderCustomFields = (log: LogItem, isExpanded: boolean, maxLength: number) => {
    const fieldKeys = Object.keys(log.fields || {});

    if (fieldKeys.length === 0) {
      return <span className="text-xs text-gray-400">-</span>;
    }

    // Create a string representation of all fields
    const fieldsString = fieldKeys.map((key) => `${key}: ${log.fields?.[key]}`).join(', ');

    const { text: displayText, isTruncated } = isExpanded
      ? { text: fieldsString, isTruncated: false }
      : truncateText(fieldsString, maxLength);

    if (fieldsString.length === 0) {
      return <span className="text-xs text-gray-400">-</span>;
    }

    return (
      <div className="space-y-1">
        {isExpanded ? (
          fieldKeys.map((key) => {
            const { formatted, isJson } = formatFieldValue(log.fields?.[key] || '');
            return (
              <div key={key} className="text-xs">
                <span className="font-medium text-gray-700">{key}:</span>{' '}
                <span
                  className={`font-mono text-gray-600 ${
                    isJson || formatted.includes(' ') ? 'whitespace-pre-wrap' : ''
                  }`}
                >
                  {formatted}
                </span>
              </div>
            );
          })
        ) : (
          <div className="text-xs">
            <span className="font-mono text-gray-600">{displayText}</span>
            {isTruncated && (
              <span className="ml-1 cursor-pointer text-emerald-600 hover:text-emerald-700">
                (expand)
              </span>
            )}
          </div>
        )}
      </div>
    );
  };

  // useEffect hooks
  useEffect(() => {
    if (!isExecuting) {
      isLoadingMore.current = false;
    }
  }, [isExecuting]);

  useEffect(() => {
    const scrollContainer = containerRef.current?.closest('.overflow-y-auto') as HTMLElement;
    if (!scrollContainer) {
      return;
    }

    scrollContainer.addEventListener('scroll', handleScroll);
    return () => scrollContainer.removeEventListener('scroll', handleScroll);
  }, [handleScroll]);

  // Save message length to localStorage whenever it changes
  useEffect(() => {
    if (typeof window !== 'undefined') {
      try {
        localStorage.setItem(STORAGE_KEY, messageLength.toString());
      } catch (error) {
        console.warn('Failed to save message length to localStorage:', error);
      }
    }
  }, [messageLength]);

  if (!hasExecuted) {
    return null;
  }

  return (
    <div ref={containerRef} className="w-full rounded-lg border border-gray-200 bg-white shadow-sm">
      <div className="border-b border-gray-200 px-4 py-2">
        <div className="flex items-center justify-between">
          <h3 className="text-base font-medium text-gray-900">Query Results</h3>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2">
              <label htmlFor="messageLength" className="text-xs font-normal text-gray-500">
                Message length:
              </label>
              <input
                id="messageLength"
                type="number"
                value={messageLength}
                onChange={(e) =>
                  setMessageLength(
                    Math.max(10, parseInt(e.target.value) || getStoredMessageLength()),
                  )
                }
                className="w-16 rounded border border-gray-300 px-1 py-0.5 text-xs focus:border-emerald-600 focus:ring-1 focus:ring-emerald-600 focus:outline-none"
                min="10"
                max="1000"
              />
            </div>
            <span className="text-xs font-normal text-gray-500">
              {isExecuting && queryResults.length === 0 ? (
                <Spin indicator={<LoadingOutlined spin />} size="small" />
              ) : (
                `${queryResults.length}${totalResults > queryResults.length ? `+ of ${totalResults}` : ''} results${queryResults.length > 0 ? ' loaded' : ' found'}`
              )}
            </span>
          </div>
        </div>
      </div>

      <div className="p-3">
        {isExecuting && queryResults.length === 0 ? (
          <div className="flex h-32 items-center justify-center">
            <Spin indicator={<LoadingOutlined spin />} />
            <span className="ml-2 text-sm">Executing query...</span>
          </div>
        ) : queryResults.length === 0 ? (
          <div className="flex h-20 items-center justify-center text-sm text-gray-500">
            No logs found matching your query.
          </div>
        ) : (
          <div className="space-y-1">
            {/* Header Row */}
            <div className="flex gap-2 border-b border-gray-200 pb-1 text-xs font-medium text-gray-700">
              <div style={{ width: '150px' }}>Timestamp</div>
              <div style={{ width: '85px' }}>Level</div>
              <div className="flex-1">Message</div>
              <div style={{ width: '10px' }} />
              <div className="flex-1">Fields</div>
            </div>

            {/* Results Rows */}
            {queryResults.map((log) => {
              const isExpanded = expandedRows.has(log.id);
              const { text: displayMessage, isTruncated: messageIsTruncated } = isExpanded
                ? { text: log.message, isTruncated: false }
                : truncateText(log.message, messageLength);

              return (
                <div
                  key={log.id}
                  className="flex cursor-pointer items-start gap-2 border-b border-gray-100 py-1 text-xs hover:bg-gray-50"
                  onClick={() => toggleRowExpansion(log.id)}
                >
                  <div style={{ width: '150px' }} className="font-mono text-xs text-gray-600">
                    {dayjs(log.timestamp).format('MMM D HH:mm:ss.SSS')}
                  </div>
                  <div style={{ width: '85px' }}>{renderLogLevel(log.level)}</div>
                  <div
                    className={`flex-1 font-mono text-xs break-all text-gray-900 ${
                      isExpanded && displayMessage.includes(' ') ? 'whitespace-pre-wrap' : ''
                    }`}
                  >
                    {displayMessage}
                    {messageIsTruncated && !isExpanded && (
                      <span className="ml-1 text-emerald-600 hover:text-emerald-700">(expand)</span>
                    )}
                  </div>
                  <div style={{ width: '10px' }} />
                  <div className="flex-1">{renderCustomFields(log, isExpanded, messageLength)}</div>
                </div>
              );
            })}

            {/* Loading indicator for infinite scroll */}
            {isExecuting && queryResults.length > 0 && (
              <div className="flex justify-center py-2">
                <Spin indicator={<LoadingOutlined spin />} size="small" />
                <span className="ml-2 text-xs text-gray-500">Loading more results...</span>
              </div>
            )}

            {/* End of results indicator */}
            {!hasMoreResults && queryResults.length > 0 && (
              <div className="py-2 text-center text-xs text-gray-500">
                All {totalResults} results loaded
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
