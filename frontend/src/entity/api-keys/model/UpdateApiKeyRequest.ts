import type { ApiKeyStatus } from './ApiKeyStatus';

export interface UpdateApiKeyRequest {
  name?: string;
  status?: ApiKeyStatus;
}
