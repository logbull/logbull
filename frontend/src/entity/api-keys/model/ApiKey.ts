import type { ApiKeyStatus } from './ApiKeyStatus';

export interface ApiKey {
  id: string;
  name: string;
  projectId: string;
  tokenPrefix: string;
  status: ApiKeyStatus;
  createdAt: Date;
  token?: string; // Only populated during creation
}
