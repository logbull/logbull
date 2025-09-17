import type { LogLevel } from './LogLevel';

export interface LogItem {
  id: string;
  projectId: string;
  timestamp: string;
  level: LogLevel;
  message: string;
  fields?: Record<string, string>;
  clientIp?: string;
  createdAt: string;
}
