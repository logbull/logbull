import { ProjectRole } from '../../users/model/ProjectRole';

export interface ProjectMemberResponse {
  id: string;
  userId: string;
  email: string;
  role: ProjectRole;
  createdAt: Date;
}
