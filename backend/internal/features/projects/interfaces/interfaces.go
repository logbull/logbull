package projects_interfaces

import "github.com/google/uuid"

type ProjectDeletionListener interface {
	OnBeforeProjectDeletion(projectID uuid.UUID) error
}
