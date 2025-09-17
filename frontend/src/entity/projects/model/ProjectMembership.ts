import { ProjectRole } from '../../users/model/ProjectRole';

export interface ProjectMembership {
  id: string;
  userId: string;
  projectId: string;
  role: ProjectRole;
  createdAt: Date;
}
