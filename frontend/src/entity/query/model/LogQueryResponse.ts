import type { LogItem } from './LogItem';

export interface LogQueryResponse {
  logs: LogItem[];
  total: number;
  limit: number;
  offset: number;
  executedIn: string;
}
