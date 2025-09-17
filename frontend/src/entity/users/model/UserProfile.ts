import type { UserRole } from './UserRole';

export interface UserProfile {
  id: string;
  email: string;
  role: UserRole;
  isActive: boolean;
  createdAt: string;
}
