import { LoadingOutlined } from '@ant-design/icons';
import { App, Button, Spin, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import GitHubButton from 'react-github-btn';

import { APP_VERSION } from '../../../constants';
import { type DiskUsage, diskApi } from '../../../entity/disk';
import { type ProjectResponse, projectApi } from '../../../entity/projects';
import {
  type UserProfile,
  UserRole,
  type UsersSettings,
  settingsApi,
  userApi,
} from '../../../entity/users';
import {
  CreateProjectDialogComponent,
  ProjectApiKeysComponent,
  ProjectMembershipComponent,
  ProjectSettingsComponent,
} from '../../../features/projects';
import { QueryComponentComponent } from '../../../features/query';
import { SettingsComponent } from '../../../features/settings';
import { ProfileComponent } from '../../../features/users/ui/ProfileComponent';
import { UsersComponent } from '../../../features/users/ui/UsersComponent';
import { useScreenHeight } from '../../../shared/hooks';
import { ProjectSelectionComponent } from './ProjectSelectionComponent';

export const MainScreenComponent = () => {
  const { message } = App.useApp();
  const screenHeight = useScreenHeight();
  const contentHeight = screenHeight - 95;

  const [selectedTab, setSelectedTab] = useState<
    'profile' | 'logbull-settings' | 'users' | 'search' | 'settings' | 'api-keys' | 'members'
  >('search');
  const [diskUsage, setDiskUsage] = useState<DiskUsage | undefined>(undefined);
  const [user, setUser] = useState<UserProfile | undefined>(undefined);
  const [globalSettings, setGlobalSettings] = useState<UsersSettings | undefined>(undefined);

  const [projects, setProjects] = useState<ProjectResponse[]>([]);
  const [selectedProject, setSelectedProject] = useState<ProjectResponse | undefined>(undefined);

  const [isLoading, setIsLoading] = useState(false);
  const [showCreateProjectDialog, setShowCreateProjectDialog] = useState(false);

  const loadData = async () => {
    setIsLoading(true);

    try {
      const [diskUsage, user, projects, settings] = await Promise.all([
        diskApi.getDiskUsage(),
        userApi.getCurrentUser(),
        projectApi.getProjects(),
        settingsApi.getSettings(),
      ]);

      setDiskUsage(diskUsage);
      setUser(user);
      setProjects(projects.projects);
      setGlobalSettings(settings);
    } catch (e) {
      message.error((e as Error).message);
    }

    setIsLoading(false);
  };

  useEffect(() => {
    loadData();
  }, []);

  // Set selected project if none selected and projects available
  useEffect(() => {
    if (!selectedProject && projects.length > 0) {
      const previouslySelectedProjectId = localStorage.getItem('selected_project_id');
      const previouslySelectedProject = projects.find(
        (project) => project.id === previouslySelectedProjectId,
      );
      const projectToSelect = previouslySelectedProject || projects[0];
      setSelectedProject(projectToSelect);
    }
  }, [projects, selectedProject]);

  // Save selected project to localStorage
  useEffect(() => {
    if (selectedProject) {
      localStorage.setItem('selected_project_id', selectedProject.id);
    }
  }, [selectedProject]);

  const handleCreateProject = () => {
    setShowCreateProjectDialog(true);
  };

  const handleProjectCreated = async (newProject: ProjectResponse) => {
    // Reload projects and select the created one
    try {
      const projectsResponse = await projectApi.getProjects();
      setProjects(projectsResponse.projects);
      setSelectedProject(newProject);
      setSelectedTab('search');
    } catch (e) {
      message.error((e as Error).message);
    }
  };

  const isUsedMoreThan95Percent =
    diskUsage && diskUsage.usedSpaceBytes / diskUsage.totalSpaceBytes > 0.95;

  return (
    <div style={{ height: screenHeight }} className="bg-[#f5f5f5] p-3">
      {/* ===================== NAVBAR ===================== */}
      <div className="mb-3 flex h-[60px] items-center rounded bg-white p-3 shadow">
        <div className="flex items-center gap-3 hover:opacity-80">
          <a href="https://logbull.com" target="_blank" rel="noreferrer">
            <img className="h-[35px] w-[35px]" src="/logo.svg" />
          </a>
        </div>

        <div className="ml-6">
          {!isLoading && (
            <ProjectSelectionComponent
              projects={projects}
              selectedProject={selectedProject}
              onCreateProject={handleCreateProject}
              onProjectSelect={setSelectedProject}
            />
          )}
        </div>

        <div className="mr-3 ml-auto flex items-center gap-5">
          <a
            className="!text-black hover:opacity-80"
            href="https://t.me/logbull_community"
            target="_blank"
            rel="noreferrer"
          >
            Community
          </a>

          <div className="mt-1">
            <GitHubButton
              href="https://github.com/logbull/logbull"
              data-icon="octicon-star"
              data-size="large"
              data-show-count="true"
              aria-label="Star Log Bull on GitHub"
            >
              &nbsp;Star Log Bull on GitHub
            </GitHubButton>
          </div>

          {isUsedMoreThan95Percent && diskUsage && (
            <Tooltip title="To make backups locally and restore them, you need to have enough space on your disk. For restore, you need to have same amount of space that the backup size.">
              <div
                className={`cursor-pointer text-center text-xs ${isUsedMoreThan95Percent ? 'text-red-500' : 'text-gray-500'}`}
              >
                {(diskUsage.usedSpaceBytes / 1024 ** 3).toFixed(1)} of{' '}
                {(diskUsage.totalSpaceBytes / 1024 ** 3).toFixed(1)} GB
                <br />
                ROM used (
                {((diskUsage.usedSpaceBytes / diskUsage.totalSpaceBytes) * 100).toFixed(1)}%)
              </div>
            </Tooltip>
          )}
        </div>
      </div>
      {/* ===================== END NAVBAR ===================== */}

      {isLoading ? (
        <div className="flex items-center justify-center py-2" style={{ height: contentHeight }}>
          <Spin indicator={<LoadingOutlined spin />} size="large" />
        </div>
      ) : (
        <div className="relative flex">
          <div
            className="max-w-[60px] min-w-[60px] rounded bg-white py-2 shadow"
            style={{ height: contentHeight }}
          >
            {[
              {
                text: 'Search',
                name: 'search',
                icon: '/icons/menu/search-gray.svg',
                selectedIcon: '/icons/menu/search-white.svg',
                onClick: () => setSelectedTab('search'),
                isAdminOnly: false,
                marginTop: '0px',
              },
              {
                text: 'Settings',
                name: 'settings',
                icon: '/icons/menu/project-settings-gray.svg',
                selectedIcon: '/icons/menu/project-settings-white.svg',
                onClick: () => setSelectedTab('settings'),
                isAdminOnly: false,
                marginTop: '0px',
              },
              {
                text: 'Members',
                name: 'members',
                icon: '/icons/menu/users-gray.svg',
                selectedIcon: '/icons/menu/users-white.svg',
                onClick: () => setSelectedTab('members'),
                isAdminOnly: false,
                marginTop: '0px',
              },
              {
                text: 'API Keys',
                name: 'api-keys',
                icon: '/icons/menu/key-gray.svg',
                selectedIcon: '/icons/menu/key-white.svg',
                onClick: () => setSelectedTab('api-keys'),
                isAdminOnly: false,
                marginTop: '0px',
              },
              {
                text: 'Profile',
                name: 'profile',
                icon: '/icons/menu/profile-gray.svg',
                selectedIcon: '/icons/menu/profile-white.svg',
                onClick: () => setSelectedTab('profile'),
                isAdminOnly: false,
                marginTop: '0px',
              },
              {
                text: 'LogBull settings',
                name: 'logbull-settings',
                icon: '/icons/menu/global-settings-gray.svg',
                selectedIcon: '/icons/menu/global-settings-white.svg',
                onClick: () => setSelectedTab('logbull-settings'),
                isAdminOnly: true,
                marginTop: '25px',
              },
              {
                text: 'Users',
                name: 'users',
                icon: '/icons/menu/user-card-gray.svg',
                selectedIcon: '/icons/menu/user-card-white.svg',
                onClick: () => setSelectedTab('users'),
                isAdminOnly: true,
                marginTop: '0px',
              },
            ]
              .filter((tab) => !tab.isAdminOnly || user?.role === UserRole.ADMIN)
              .map((tab) => (
                <div key={tab.text} className="flex justify-center">
                  <div
                    className={`flex h-[50px] w-[50px] cursor-pointer items-center justify-center rounded ${selectedTab === tab.name ? 'bg-emerald-600' : 'hover:bg-blue-50'}`}
                    onClick={tab.onClick}
                    style={{ marginTop: tab.marginTop }}
                  >
                    <div className="mb-1">
                      <div className="flex justify-center">
                        <img
                          src={selectedTab === tab.name ? tab.selectedIcon : tab.icon}
                          width={20}
                          alt={tab.text}
                          loading="lazy"
                        />
                      </div>
                    </div>
                  </div>
                </div>
              ))}
          </div>

          {selectedTab === 'profile' && <ProfileComponent contentHeight={contentHeight} />}

          {selectedTab === 'logbull-settings' && (
            <SettingsComponent contentHeight={contentHeight} />
          )}

          {selectedTab === 'users' && <UsersComponent contentHeight={contentHeight} />}

          {projects.length === 0 &&
          (selectedTab === 'search' ||
            selectedTab === 'settings' ||
            selectedTab === 'api-keys' ||
            selectedTab === 'members') ? (
            <div
              className="flex grow items-center justify-center rounded pl-5"
              style={{ height: contentHeight }}
            >
              <Button
                type="primary"
                size="large"
                onClick={handleCreateProject}
                className="border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700"
              >
                Create project
              </Button>
            </div>
          ) : (
            <>
              {selectedTab === 'settings' && selectedProject && user && (
                <ProjectSettingsComponent
                  projectResponse={selectedProject}
                  contentHeight={contentHeight}
                  user={user}
                />
              )}
              {selectedTab === 'api-keys' && selectedProject && user && (
                <ProjectApiKeysComponent
                  projectResponse={selectedProject}
                  contentHeight={contentHeight}
                  user={user}
                />
              )}
              {selectedTab === 'members' && selectedProject && user && (
                <ProjectMembershipComponent
                  projectResponse={selectedProject}
                  contentHeight={contentHeight}
                  user={user}
                />
              )}
              {selectedTab === 'search' && selectedProject && user && (
                <QueryComponentComponent
                  projectId={selectedProject.id}
                  contentHeight={contentHeight}
                />
              )}
            </>
          )}

          <div className="absolute bottom-1 left-2 mb-[0px] text-sm text-gray-400">
            v{APP_VERSION}
          </div>
        </div>
      )}

      {/* Create Project Dialog */}
      {showCreateProjectDialog && user && globalSettings && (
        <CreateProjectDialogComponent
          user={user}
          globalSettings={globalSettings}
          onClose={() => setShowCreateProjectDialog(false)}
          onProjectCreated={handleProjectCreated}
        />
      )}
    </div>
  );
};
