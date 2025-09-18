import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { GetQueryableFieldsRequest } from '../model/GetQueryableFieldsRequest';
import type { GetQueryableFieldsResponse } from '../model/GetQueryableFieldsResponse';
import type { LogQueryRequest } from '../model/LogQueryRequest';
import type { LogQueryResponse } from '../model/LogQueryResponse';
import type { ProjectLogStats } from '../model/ProjectLogStats';

export const queryApi = {
  async executeQuery(projectId: string, request: LogQueryRequest): Promise<LogQueryResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/logs/query/execute/${projectId}`,
      requestOptions,
    );
  },

  async getQueryableFields(
    projectId: string,
    request?: GetQueryableFieldsRequest,
  ): Promise<GetQueryableFieldsResponse> {
    const requestOptions: RequestOptions = new RequestOptions();

    let url = `${getApplicationServer()}/api/v1/logs/query/fields/${projectId}`;
    if (request?.query) {
      const searchParams = new URLSearchParams({ query: request.query });
      url += `?${searchParams.toString()}`;
    }

    return apiHelper.fetchGetJson(url, requestOptions);
  },

  async getProjectStats(projectId: string): Promise<ProjectLogStats> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson(
      `${getApplicationServer()}/api/v1/logs/query/stats/${projectId}`,
      requestOptions,
    );
  },
};
