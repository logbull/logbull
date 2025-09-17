export interface AuditLog {
  id: string;
  userId?: string;
  projectId?: string;
  message: string;
  createdAt: string;
  userEmail?: string;
  projectName?: string;
}
