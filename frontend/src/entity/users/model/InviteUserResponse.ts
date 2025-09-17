import type { ProjectRole } from './ProjectRole';

export interface InviteUserResponse {
  id: string;
  email: string;
  intendedProjectId?: string;
  intendedProjectRole?: ProjectRole;
  createdAt: string;
}
