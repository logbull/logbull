package projects_repositories

import (
	"time"

	projects_models "logbull/internal/features/projects/models"
	"logbull/internal/storage"

	"github.com/google/uuid"
)

type ProjectRepository struct{}

func (r *ProjectRepository) CreateProject(project *projects_models.Project) error {
	if project.ID == uuid.Nil {
		project.ID = uuid.New()
	}
	if project.CreatedAt.IsZero() {
		project.CreatedAt = time.Now().UTC()
	}

	return storage.GetDb().Create(project).Error
}

func (r *ProjectRepository) GetProjectByID(projectID uuid.UUID) (*projects_models.Project, error) {
	var project projects_models.Project

	if err := storage.GetDb().Where("id = ?", projectID).First(&project).Error; err != nil {
		return nil, err
	}

	return &project, nil
}

func (r *ProjectRepository) UpdateProject(project *projects_models.Project) error {
	return storage.GetDb().Save(project).Error
}

func (r *ProjectRepository) DeleteProject(projectID uuid.UUID) error {
	return storage.GetDb().Delete(&projects_models.Project{}, projectID).Error
}

func (r *ProjectRepository) GetAllProjects() ([]*projects_models.Project, error) {
	var projects []*projects_models.Project

	err := storage.GetDb().Order("created_at DESC").Find(&projects).Error

	return projects, err
}
