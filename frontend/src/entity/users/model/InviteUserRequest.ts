import type { ProjectRole } from './ProjectRole';

export interface InviteUserRequest {
  email: string;
  intendedProjectId?: string;
  intendedProjectRole?: ProjectRole;
}
