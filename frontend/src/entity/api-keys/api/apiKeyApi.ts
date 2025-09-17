import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { ApiKey } from '../model/ApiKey';
import type { CreateApiKeyRequest } from '../model/CreateApiKeyRequest';
import type { GetApiKeysResponse } from '../model/GetApiKeysResponse';
import type { UpdateApiKeyRequest } from '../model/UpdateApiKeyRequest';
import type { ValidateTokenRequest } from '../model/ValidateTokenRequest';
import type { ValidateTokenResponse } from '../model/ValidateTokenResponse';

export const apiKeyApi = {
  async createApiKey(projectId: string, request: CreateApiKeyRequest): Promise<ApiKey> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/projects/api-keys/${projectId}`,
      requestOptions,
    );
  },

  async getApiKeys(projectId: string): Promise<GetApiKeysResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson(
      `${getApplicationServer()}/api/v1/projects/api-keys/${projectId}`,
      requestOptions,
    );
  },

  async updateApiKey(
    projectId: string,
    apiKeyId: string,
    request: UpdateApiKeyRequest,
  ): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPutJson(
      `${getApplicationServer()}/api/v1/projects/api-keys/${projectId}/${apiKeyId}`,
      requestOptions,
    );
  },

  async deleteApiKey(projectId: string, apiKeyId: string): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchDeleteJson(
      `${getApplicationServer()}/api/v1/projects/api-keys/${projectId}/${apiKeyId}`,
      requestOptions,
    );
  },

  async validateToken(request: ValidateTokenRequest): Promise<ValidateTokenResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/validate-token`,
      requestOptions,
    );
  },
};
