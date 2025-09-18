import { LoadingOutlined, PlayCircleOutlined } from '@ant-design/icons';
import { App, Button, Divider, Spin, Switch } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import {
  type GetQueryableFieldsRequest,
  type LogItem,
  type LogQueryRequest,
  type QueryNode,
  type QueryableField,
  queryApi,
} from '../../../entity/query';
import { FloatingTopButtonComponent } from './FloatingTopButtonComponent';
import { HowToSendLogsFromCodeComponent } from './HowToSendLogsFromCodeComponent';
import { QueryBuilderComponent } from './QueryBuilderComponent';
import { QueryResultsComponent } from './QueryResultsComponent';
import { type TimeRange, TimeRangePickerComponent } from './TimeRangePickerComponent';

interface Props {
  projectId: string;
  contentHeight: number;
}

/**
 * QueryComponent - A comprehensive log query builder and results viewer
 *
 * Features:
 * - Visual query builder supporting complex nested conditions
 * - Support for all query operators (equals, contains, in, exists, etc.)
 * - Logical operators (AND, OR, NOT) with unlimited nesting
 * - Dynamic field discovery from backend
 * - Time range filtering with date/time picker
 * - Sort order control (ascending/descending by timestamp)
 * - Results table with pagination
 * - Proper TypeScript typing throughout
 * - Responsive design with Ant Design components
 *
 * Query Structure:
 * - Simple conditions: field + operator + value
 * - Logical groups: operator + array of child conditions/groups
 * - Unlimited nesting depth (limited by backend validation)
 *
 * Supported Field Types:
 * - Standard fields: message, level, client_ip, timestamp
 * - Custom fields: any user-defined fields with flexible naming
 *
 * Backend Integration:
 * - Fetches available fields via /api/v1/logs/query/fields/{projectId}
 * - Executes queries via /api/v1/logs/query/execute/{projectId}
 * - Handles query validation and error responses
 */

interface SavedQuery {
  query: QueryNode | null;
  sortOrder: 'asc' | 'desc';
}

export const QueryComponentComponent = ({ projectId, contentHeight }: Props): React.JSX.Element => {
  // States
  const [isShowHowToSendLogsFromCode, setIsShowHowToSendLogsFromCode] = useState(false);
  const [queryableFields, setQueryableFields] = useState<QueryableField[]>([]);
  const [isSearchingFields, setIsSearchingFields] = useState(false);
  const [currentQuery, setCurrentQuery] = useState<QueryNode | null>(null);
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');
  const [isExecuting, setIsExecuting] = useState(false);
  const [queryResults, setQueryResults] = useState<LogItem[]>([]);
  const [totalResults, setTotalResults] = useState(0);
  const [hasExecuted, setHasExecuted] = useState(false);
  const [hasMoreResults, setHasMoreResults] = useState(false);
  const [frozenTimeRange, setFrozenTimeRange] = useState<TimeRange | null>(null);
  const [pageSize] = useState(200);
  const [hasSearched, setHasSearched] = useState(false);
  const [isInitialLoad, setIsInitialLoad] = useState(true);

  // Refs
  const timeRangeRef = useRef<() => TimeRange | null>(null);
  const timeRangeHelpersRef = useRef<{
    isUntilNow: () => boolean;
    refreshRange: () => void;
  } | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const queryBuilderRef = useRef<HTMLDivElement>(null);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Functions
  const { message } = App.useApp();

  // Query persistence functions
  const getSavedQueryKey = (projectId: string): string => {
    return `logbull-query-${projectId}`;
  };

  const saveQueryToStorage = (query: QueryNode | null, sortOrder: 'asc' | 'desc') => {
    try {
      const savedQuery: SavedQuery = {
        query,
        sortOrder,
      };
      localStorage.setItem(getSavedQueryKey(projectId), JSON.stringify(savedQuery));
    } catch (error) {
      console.warn('Failed to save query to localStorage:', error);
    }
  };

  const loadQueryFromStorage = (): SavedQuery | null => {
    try {
      const saved = localStorage.getItem(getSavedQueryKey(projectId));
      if (saved) {
        return JSON.parse(saved) as SavedQuery;
      }
    } catch (error) {
      console.warn('Failed to load query from localStorage:', error);
    }
    return null;
  };

  const loadQueryableFields = async (searchTerm?: string, isSearch = false) => {
    if (isSearch) {
      setIsSearchingFields(true);
    }

    try {
      const request: GetQueryableFieldsRequest | undefined = searchTerm
        ? { query: searchTerm }
        : undefined;
      const response = await queryApi.getQueryableFields(projectId, request);
      setQueryableFields(response.fields);
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to load queryable fields';
      message.error(errorMessage);
    } finally {
      if (isSearch) {
        setIsSearchingFields(false);
      }
    }
  };

  const debouncedLoadFields = useCallback(
    (searchTerm?: string) => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      timeoutRef.current = setTimeout(() => loadQueryableFields(searchTerm, true), 250);
    },
    [projectId],
  );

  // Helper function to check if operator needs value input
  const operatorNeedsValue = (operator: string): boolean => {
    return operator !== 'exists' && operator !== 'not_exists';
  };

  // Helper function to check if operator expects array input
  const operatorExpectsArray = (operator: string): boolean => {
    return operator === 'in' || operator === 'not_in';
  };

  // Validate query for empty fields and missing required values
  const validateQuery = (query: QueryNode | null): { isValid: boolean; error?: string } => {
    if (!query) {
      return { isValid: true }; // Empty query is valid (shows all logs)
    }

    const checkEmptyFields = (node: QueryNode): boolean => {
      if (node.type === 'condition' && node.condition) {
        const field = node.condition.field;
        return !field || field.trim() === '';
      }

      if (node.type === 'logical' && node.logic) {
        return node.logic.children.some(checkEmptyFields);
      }

      return false;
    };

    const checkMissingValues = (node: QueryNode): boolean => {
      if (node.type === 'condition' && node.condition) {
        const { operator, value } = node.condition;

        // Check if this operator needs a value
        if (operatorNeedsValue(operator)) {
          // For array operators (in, not_in), check if array is empty or undefined
          if (operatorExpectsArray(operator)) {
            return !Array.isArray(value) || value.length === 0;
          }

          // For non-array operators, check if value is empty, null, or undefined
          return value === null || value === undefined || value === '';
        }

        return false;
      }

      if (node.type === 'logical' && node.logic) {
        return node.logic.children.some(checkMissingValues);
      }

      return false;
    };

    if (checkEmptyFields(query)) {
      return {
        isValid: false,
        error: 'Please fill in all field names before executing the query.',
      };
    }

    if (checkMissingValues(query)) {
      return {
        isValid: false,
        error:
          'Please provide values for all conditions that require them before executing the query.',
      };
    }

    return { isValid: true };
  };

  const executeQuery = async (isLoadMore = false) => {
    // Validate query before execution (only for new queries, not load more)
    if (!isLoadMore) {
      const validation = validateQuery(currentQuery);
      if (!validation.isValid) {
        message.error(validation.error);
        return;
      }
    }

    setIsExecuting(true);
    try {
      const request: LogQueryRequest = {
        query: currentQuery, // Send null when no query is built
        limit: pageSize,
        offset: isLoadMore ? queryResults.length : 0,
        sortOrder,
      };

      // For new queries, get fresh time range. For load more, use frozen time range
      let timeRangeToUse: TimeRange | null = null;
      if (isLoadMore && frozenTimeRange) {
        timeRangeToUse = frozenTimeRange;
      } else {
        const currentTimeRange = timeRangeRef.current?.();
        timeRangeToUse = currentTimeRange || null;
        // Freeze the time range for subsequent load more operations
        if (timeRangeToUse) {
          setFrozenTimeRange(timeRangeToUse);
        }
      }

      if (timeRangeToUse) {
        request.timeRange = {
          from: timeRangeToUse.from.toISOString(),
          to: timeRangeToUse.to.toISOString(),
        };
      }

      const response = await queryApi.executeQuery(projectId, request);

      if (isLoadMore) {
        // Append new results to existing ones
        setQueryResults((prev) => [...prev, ...response.logs]);
      } else {
        // Replace results for new query
        setQueryResults(response.logs);
        setTotalResults(response.total);
        setHasExecuted(true);
      }

      // Check if there are more results to load
      const currentResultsCount = isLoadMore
        ? queryResults.length + response.logs.length
        : response.logs.length;
      setHasMoreResults(currentResultsCount < response.total);

      if (!isLoadMore) {
        const queryType = currentQuery ? 'matching your query' : '(showing all logs)';
        message.success(`Found ${response.total} logs ${queryType} (${response.executedIn})`);
        setHasSearched(true);
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Query execution failed';
      message.error(errorMessage);
    } finally {
      setIsExecuting(false);
    }
  };

  const handleLoadMore = () => {
    executeQuery(true);
  };

  const handleExecuteOrRefresh = async () => {
    if (hasSearched) {
      // If we've already searched, check if we can refresh the time range
      const helpers = timeRangeHelpersRef.current;
      if (helpers?.isUntilNow()) {
        // Refresh the time range to update "now" and then execute query
        helpers.refreshRange();
        // Reset hasSearched to false so the new query execution will be treated as fresh
        setHasSearched(false);
        // Execute after a small delay to ensure the range has been updated
        setTimeout(() => executeQuery(false), 50);
      } else {
        // For custom ranges, just re-execute with the same range
        executeQuery(false);
      }
    } else {
      // First time execution
      executeQuery(false);
    }
  };

  // useEffect hooks
  useEffect(() => {
    loadQueryableFields();

    // Load saved query for this project
    const savedQuery = loadQueryFromStorage();
    if (savedQuery) {
      setCurrentQuery(savedQuery.query);
      setSortOrder(savedQuery.sortOrder);
    } else {
      // Reset to defaults for new project
      setCurrentQuery(null);
      setSortOrder('desc');
    }

    // Reset other states when switching projects
    setQueryResults([]);
    setTotalResults(0);
    setHasExecuted(false);
    setHasMoreResults(false);
    setFrozenTimeRange(null);
    setHasSearched(false);

    // Mark initial load as complete
    setIsInitialLoad(false);
  }, [projectId]);

  // Save query and sort order whenever they change (but not on initial load)
  useEffect(() => {
    if (!isInitialLoad) {
      saveQueryToStorage(currentQuery, sortOrder);
    }
  }, [currentQuery, sortOrder, projectId, isInitialLoad]);

  return (
    <div
      ref={containerRef}
      className="ml-3 w-full space-y-3 overflow-y-auto"
      style={{ height: contentHeight }}
    >
      <FloatingTopButtonComponent containerRef={containerRef} />

      {/* Query Builder Section */}
      <div
        ref={queryBuilderRef}
        className="w-full rounded-lg border border-gray-200 bg-white shadow-sm"
      >
        <div className="flex items-center px-6 py-4">
          <TimeRangePickerComponent
            onChange={() => {
              setHasSearched(false);
            }}
            onGetCurrentRange={(getCurrentRange: () => TimeRange | null) => {
              timeRangeRef.current = getCurrentRange;
            }}
            onGetRangeHelpers={(helpers) => {
              timeRangeHelpersRef.current = helpers;
            }}
          />

          <div className="ml-5">
            <label className="mb-1 block text-sm font-medium text-gray-700">Sort Order</label>
            <div className="flex items-center gap-2">
              <span
                className={`text-sm ${sortOrder === 'desc' ? 'text-gray-900' : 'text-gray-400'}`}
              >
                Newest first
              </span>
              <Switch
                checked={sortOrder === 'asc'}
                onChange={(checked) => {
                  setSortOrder(checked ? 'asc' : 'desc');
                  setHasSearched(false);
                }}
                size="small"
              />
              <span
                className={`text-sm ${sortOrder === 'asc' ? 'text-gray-900' : 'text-gray-400'}`}
              >
                Oldest first
              </span>
            </div>
          </div>

          <div className="ml-auto">
            <Button
              type="primary"
              onClick={() => setIsShowHowToSendLogsFromCode(!isShowHowToSendLogsFromCode)}
              loading={isExecuting}
              ghost
              className="border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700"
            >
              How to send logs from code?
            </Button>
          </div>
        </div>

        <div className="space-y-4 p-6">
          <QueryBuilderComponent
            fields={queryableFields}
            query={currentQuery}
            onChange={(query) => {
              setCurrentQuery(query);
              setHasSearched(false);
            }}
            onFieldSearch={debouncedLoadFields}
            isSearchingFields={isSearchingFields}
          />

          <Divider />

          {/* Execution Controls */}
          <div className="flex items-center justify-between">
            {isExecuting ? (
              <Spin indicator={<LoadingOutlined spin />} />
            ) : (
              <Button
                type="primary"
                icon={<PlayCircleOutlined />}
                onClick={handleExecuteOrRefresh}
                size="large"
                ghost={hasSearched}
                className={`ml-auto ${
                  hasSearched
                    ? 'border-emerald-600 text-emerald-600 hover:border-emerald-700 hover:text-emerald-700'
                    : 'border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700'
                }`}
              >
                {hasSearched ? 'Refresh Query' : 'Execute Query'}
              </Button>
            )}
          </div>
        </div>
      </div>

      {/* Results Section */}
      <QueryResultsComponent
        queryResults={queryResults}
        totalResults={totalResults}
        hasExecuted={hasExecuted}
        isExecuting={isExecuting}
        hasMoreResults={hasMoreResults}
        onLoadMore={handleLoadMore}
      />

      {isShowHowToSendLogsFromCode && (
        <HowToSendLogsFromCodeComponent
          projectId={projectId}
          onClose={() => setIsShowHowToSendLogsFromCode(false)}
        />
      )}
    </div>
  );
};
