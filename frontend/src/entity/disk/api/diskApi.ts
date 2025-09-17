import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { DiskUsage } from '../model/DiskUsage';

export const diskApi = {
  async getDiskUsage() {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson<DiskUsage>(
      `${getApplicationServer()}/api/v1/disk/usage`,
      requestOptions,
      true,
    );
  },
};
