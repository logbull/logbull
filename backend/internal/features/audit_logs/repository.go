package audit_logs

import (
	"logbull/internal/storage"
	"time"

	"github.com/google/uuid"
)

type AuditLogRepository struct{}

func (r *AuditLogRepository) Create(auditLog *AuditLog) error {
	if auditLog.ID == uuid.Nil {
		auditLog.ID = uuid.New()
	}

	return storage.GetDb().Create(auditLog).Error
}

func (r *AuditLogRepository) GetGlobal(limit, offset int, beforeDate *time.Time) ([]*AuditLogDTO, error) {
	var auditLogs = make([]*AuditLogDTO, 0)

	sql := `
		SELECT 
			al.id,
			al.user_id,
			al.project_id,
			al.message,
			al.created_at,
			u.email as user_email,
			p.name as project_name
		FROM audit_logs al
		LEFT JOIN users u ON al.user_id = u.id
		LEFT JOIN projects p ON al.project_id = p.id`

	args := []interface{}{}

	if beforeDate != nil {
		sql += " WHERE al.created_at < ?"
		args = append(args, *beforeDate)
	}

	sql += " ORDER BY al.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	err := storage.GetDb().Raw(sql, args...).Scan(&auditLogs).Error

	return auditLogs, err
}

func (r *AuditLogRepository) GetByUser(
	userID uuid.UUID,
	limit, offset int,
	beforeDate *time.Time,
) ([]*AuditLogDTO, error) {
	var auditLogs = make([]*AuditLogDTO, 0)

	sql := `
		SELECT 
			al.id,
			al.user_id,
			al.project_id,
			al.message,
			al.created_at,
			u.email as user_email,
			p.name as project_name
		FROM audit_logs al
		LEFT JOIN users u ON al.user_id = u.id
		LEFT JOIN projects p ON al.project_id = p.id
		WHERE al.user_id = ?`

	args := []interface{}{userID}

	if beforeDate != nil {
		sql += " AND al.created_at < ?"
		args = append(args, *beforeDate)
	}

	sql += " ORDER BY al.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	err := storage.GetDb().Raw(sql, args...).Scan(&auditLogs).Error

	return auditLogs, err
}

func (r *AuditLogRepository) GetByProject(
	projectID uuid.UUID,
	limit, offset int,
	beforeDate *time.Time,
) ([]*AuditLogDTO, error) {
	var auditLogs = make([]*AuditLogDTO, 0)

	sql := `
		SELECT 
			al.id,
			al.user_id,
			al.project_id,
			al.message,
			al.created_at,
			u.email as user_email,
			p.name as project_name
		FROM audit_logs al
		LEFT JOIN users u ON al.user_id = u.id
		LEFT JOIN projects p ON al.project_id = p.id
		WHERE al.project_id = ?`

	args := []interface{}{projectID}

	if beforeDate != nil {
		sql += " AND al.created_at < ?"
		args = append(args, *beforeDate)
	}

	sql += " ORDER BY al.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	err := storage.GetDb().Raw(sql, args...).Scan(&auditLogs).Error

	return auditLogs, err
}

func (r *AuditLogRepository) CountGlobal(beforeDate *time.Time) (int64, error) {
	var count int64
	query := storage.GetDb().Model(&AuditLog{})

	if beforeDate != nil {
		query = query.Where("created_at < ?", *beforeDate)
	}

	err := query.Count(&count).Error
	return count, err
}
