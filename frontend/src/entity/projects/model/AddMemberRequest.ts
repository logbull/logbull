import { ProjectRole } from '../../users/model/ProjectRole';

export interface AddMemberRequest {
  email: string;
  role: ProjectRole;
}
