export interface Project {
  id: string;
  name: string;
  createdAt: Date;

  // Security Policies
  isApiKeyRequired: boolean;
  isFilterByDomain: boolean;
  isFilterByIp: boolean;
  allowedDomains: string[];
  allowedIps: string[];

  // Rate Limiting & Quotas
  logsPerSecondLimit: number;
  maxLogsAmount: number;
  maxLogsSizeMb: number;
  maxLogsLifeDays: number;
  maxLogSizeKb: number;
}
