import { LoadingOutlined } from '@ant-design/icons';
import { App, Button, Input, Modal } from 'antd';
import { Spin } from 'antd';
import { useState } from 'react';

import type { ProjectResponse } from '../../../entity/projects';
import { projectApi } from '../../../entity/projects';
import { type UserProfile, UserRole, type UsersSettings } from '../../../entity/users';

interface Props {
  user: UserProfile;
  globalSettings: UsersSettings;

  onClose: () => void;
  onProjectCreated: (project: ProjectResponse) => void;
}

export const CreateProjectDialogComponent = ({
  user,
  globalSettings,
  onClose,
  onProjectCreated,
}: Props) => {
  const { message } = App.useApp();
  const [isCreating, setIsCreating] = useState(false);
  const [projectName, setProjectName] = useState('');

  const isAllowedToCreateProjects =
    globalSettings.isMemberAllowedToCreateProjects || user.role === UserRole.ADMIN;

  const handleCreateProject = async () => {
    if (!projectName.trim()) {
      message.error('Please enter a project name');
      return;
    }

    setIsCreating(true);

    try {
      const newProject = await projectApi.createProject({
        name: projectName.trim(),
      });

      message.success('Project created successfully');
      onProjectCreated(newProject);
      onClose();
    } catch (error) {
      message.error((error as Error).message || 'Failed to create project');
    } finally {
      setIsCreating(false);
    }
  };

  if (!isAllowedToCreateProjects) {
    return (
      <Modal
        title="Permission denied"
        open
        onCancel={onClose}
        footer={[
          <Button key="ok" type="primary" onClick={onClose}>
            OK
          </Button>,
        ]}
      >
        <p>
          You don&apos;t have permission to create projects. Please ask the administrator to create
          the project for you.
        </p>
      </Modal>
    );
  }

  return (
    <Modal
      title="Create project"
      open
      onCancel={onClose}
      footer={[
        <Button key="cancel" onClick={onClose} disabled={isCreating}>
          Cancel
        </Button>,

        <Button
          key="create"
          type="primary"
          onClick={handleCreateProject}
          disabled={isCreating || !projectName.trim()}
          className="border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700"
        >
          {isCreating ? (
            <Spin indicator={<LoadingOutlined spin />} size="small" />
          ) : (
            'Create project'
          )}
        </Button>,
      ]}
    >
      <div className="mb-4">
        <label className="mb-2 block text-sm font-medium text-gray-700">Project name</label>
        <Input
          value={projectName}
          onChange={(e) => setProjectName(e.target.value)}
          placeholder="Enter project name"
          disabled={isCreating}
          onPressEnter={handleCreateProject}
          autoFocus
        />
      </div>
    </Modal>
  );
};
