import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { GetAuditLogsResponse } from '../../audit-logs/model/GetAuditLogsResponse';
import type { CreateProjectRequest } from '../model/CreateProjectRequest';
import type { ListProjectsResponse } from '../model/ListProjectsResponse';
import type { Project } from '../model/Project';
import type { ProjectResponse } from '../model/ProjectResponse';

export const projectApi = {
  async createProject(request: CreateProjectRequest): Promise<ProjectResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPostJson(`${getApplicationServer()}/api/v1/projects`, requestOptions);
  },

  async getProjects(): Promise<ListProjectsResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson(`${getApplicationServer()}/api/v1/projects`, requestOptions);
  },

  async getProject(projectId: string): Promise<Project> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson(
      `${getApplicationServer()}/api/v1/projects/${projectId}`,
      requestOptions,
    );
  },

  async updateProject(projectId: string, project: Project): Promise<Project> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(project));
    return apiHelper.fetchPutJson(
      `${getApplicationServer()}/api/v1/projects/${projectId}`,
      requestOptions,
    );
  },

  async deleteProject(projectId: string): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchDeleteJson(
      `${getApplicationServer()}/api/v1/projects/${projectId}`,
      requestOptions,
    );
  },

  async getProjectAuditLogs(
    projectId: string,
    params?: {
      limit?: number;
      offset?: number;
      beforeDate?: string;
    },
  ): Promise<GetAuditLogsResponse> {
    const requestOptions: RequestOptions = new RequestOptions();

    let url = `${getApplicationServer()}/api/v1/projects/${projectId}/audit-logs`;
    const urlParams = new URLSearchParams();

    if (params?.limit) {
      urlParams.append('limit', params.limit.toString());
    }
    if (params?.offset) {
      urlParams.append('offset', params.offset.toString());
    }
    if (params?.beforeDate) {
      urlParams.append('beforeDate', params.beforeDate);
    }

    if (urlParams.toString()) {
      url += `?${urlParams.toString()}`;
    }

    return apiHelper.fetchGetJson(url, requestOptions);
  },
};
