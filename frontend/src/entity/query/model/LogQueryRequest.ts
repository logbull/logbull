import type { QueryNode } from './QueryNode';
import type { TimeRange } from './TimeRange';

export interface LogQueryRequest {
  query: QueryNode | null;
  timeRange?: TimeRange;
  limit?: number;
  offset?: number;
  sortOrder?: 'asc' | 'desc';
}
