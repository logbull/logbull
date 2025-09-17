import { ProjectRole } from '../../users/model/ProjectRole';

export interface ProjectResponse {
  id: string;
  name: string;
  createdAt: Date;
  userRole?: ProjectRole;
}
