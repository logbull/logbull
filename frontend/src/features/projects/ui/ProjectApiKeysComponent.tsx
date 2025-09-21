import {
  CopyOutlined,
  DeleteOutlined,
  EditOutlined,
  LoadingOutlined,
  PlusOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import { App, Button, Input, Modal, Popconfirm, Spin, Switch, Table, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';

import {
  type ApiKey,
  type CreateApiKeyRequest,
  type UpdateApiKeyRequest,
  apiKeyApi,
} from '../../../entity/api-keys';
import { ApiKeyStatus } from '../../../entity/api-keys/model/ApiKeyStatus';
import type { ProjectResponse } from '../../../entity/projects';
import { projectApi } from '../../../entity/projects/api/projectApi';
import type { Project } from '../../../entity/projects/model/Project';
import type { UserProfile } from '../../../entity/users';
import { ProjectRole } from '../../../entity/users/model/ProjectRole';
import { UserRole } from '../../../entity/users/model/UserRole';
import { copyToClipboard } from '../../../shared/lib';

interface Props {
  contentHeight: number;
  projectResponse: ProjectResponse;
  user: UserProfile;
}

export function ProjectApiKeysComponent({ contentHeight, projectResponse, user }: Props) {
  const { message } = App.useApp();

  // Project and API keys state
  const [project, setProject] = useState<Project | undefined>(undefined);
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([]);
  const [isLoadingProject, setIsLoadingProject] = useState(true);
  const [isLoadingApiKeys, setIsLoadingApiKeys] = useState(true);

  // Create API key modal state
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [createForm, setCreateForm] = useState({ name: '' });
  const [isCreating, setIsCreating] = useState(false);
  const [createNameError, setCreateNameError] = useState(false);

  // Edit API key state
  const [editingKey, setEditingKey] = useState<{ id: string; name: string } | null>(null);
  const [editForm, setEditForm] = useState({ name: '' });
  const [isUpdating, setIsUpdating] = useState(false);
  const [editNameError, setEditNameError] = useState(false);

  // Processing states
  const [processingKeys, setProcessingKeys] = useState<Set<string>>(new Set());
  const [deletingKeys, setDeletingKeys] = useState<Set<string>>(new Set());

  // Token display modal state
  const [isTokenModalOpen, setIsTokenModalOpen] = useState(false);
  const [createdApiKey, setCreatedApiKey] = useState<ApiKey | null>(null);

  // Permissions check
  const canManageKeys =
    user.role === UserRole.ADMIN ||
    projectResponse.userRole === ProjectRole.OWNER ||
    projectResponse.userRole === ProjectRole.ADMIN;

  useEffect(() => {
    loadProjectSettings();
    loadApiKeys();
  }, [projectResponse.id]);

  const loadProjectSettings = async () => {
    setIsLoadingProject(true);
    try {
      const projectData = await projectApi.getProject(projectResponse.id);
      setProject(projectData);
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to load project settings';
      message.error(errorMessage);
    } finally {
      setIsLoadingProject(false);
    }
  };

  const loadApiKeys = async () => {
    setIsLoadingApiKeys(true);
    try {
      const response = await apiKeyApi.getApiKeys(projectResponse.id);
      setApiKeys(response.apiKeys);
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to load API keys';
      message.error(errorMessage);
    } finally {
      setIsLoadingApiKeys(false);
    }
  };

  const handleCreateApiKey = async () => {
    if (!createForm.name.trim()) {
      setCreateNameError(true);
      message.error('API key name is required');
      return;
    }
    setCreateNameError(false);
    setIsCreating(true);

    try {
      const request: CreateApiKeyRequest = { name: createForm.name.trim() };
      const newApiKey = await apiKeyApi.createApiKey(projectResponse.id, request);

      // Close the create modal first
      setCreateForm({ name: '' });
      setIsCreateModalOpen(false);

      // Show the token in a modal since it's only shown once
      if (newApiKey.token) {
        // Store the created API key and show the modal
        setCreatedApiKey(newApiKey);
        setTimeout(() => {
          setIsTokenModalOpen(true);
        }, 100); // Small delay to ensure create modal is fully closed
      } else {
        // If no token was returned, show error
        Modal.error({
          title: 'API Key Creation Issue',
          content: 'The API key was created but no token was returned. Please contact support.',
        });
      }

      setApiKeys((prev) => [newApiKey, ...prev]);
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to create API key';
      message.error(errorMessage);
    } finally {
      setIsCreating(false);
    }
  };

  const handleUpdateApiKey = async () => {
    if (!editingKey || !editForm.name.trim()) {
      setEditNameError(true);
      message.error('API key name is required');
      return;
    }
    setEditNameError(false);
    setIsUpdating(true);

    try {
      const request: UpdateApiKeyRequest = { name: editForm.name.trim() };
      await apiKeyApi.updateApiKey(projectResponse.id, editingKey.id, request);

      setApiKeys((prev) =>
        prev.map((key) =>
          key.id === editingKey.id ? { ...key, name: editForm.name.trim() } : key,
        ),
      );

      setEditingKey(null);
      setEditForm({ name: '' });
      message.success('API key updated successfully');
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to update API key';
      message.error(errorMessage);
    } finally {
      setIsUpdating(false);
    }
  };

  const handleToggleStatus = async (apiKeyId: string, currentStatus: ApiKeyStatus) => {
    const newStatus =
      currentStatus === ApiKeyStatus.ACTIVE ? ApiKeyStatus.DISABLED : ApiKeyStatus.ACTIVE;

    setApiKeys((prev) =>
      prev.map((key) => (key.id === apiKeyId ? { ...key, status: newStatus } : key)),
    );
    setProcessingKeys((prev) => new Set(prev).add(apiKeyId));

    try {
      const request: UpdateApiKeyRequest = { status: newStatus };
      await apiKeyApi.updateApiKey(projectResponse.id, apiKeyId, request);

      const statusText = newStatus === ApiKeyStatus.ACTIVE ? 'enabled' : 'disabled';
      message.success(`API key ${statusText} successfully`);
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to update API key status';
      message.error(errorMessage);

      // Revert status change on error
      const revertStatus =
        newStatus === ApiKeyStatus.ACTIVE ? ApiKeyStatus.DISABLED : ApiKeyStatus.ACTIVE;
      setApiKeys((prev) =>
        prev.map((key) => (key.id === apiKeyId ? { ...key, status: revertStatus } : key)),
      );
    } finally {
      setProcessingKeys((prev) => {
        const newSet = new Set(prev);
        newSet.delete(apiKeyId);
        return newSet;
      });
    }
  };

  const handleDeleteApiKey = async (apiKeyId: string, apiKeyName: string) => {
    setDeletingKeys((prev) => new Set(prev).add(apiKeyId));

    try {
      await apiKeyApi.deleteApiKey(projectResponse.id, apiKeyId);
      setApiKeys((prev) => prev.filter((key) => key.id !== apiKeyId));
      message.success(`API key "${apiKeyName}" deleted successfully`);
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to delete API key';
      message.error(errorMessage);
    } finally {
      setDeletingKeys((prev) => {
        const newSet = new Set(prev);
        newSet.delete(apiKeyId);
        return newSet;
      });
    }
  };

  const startEditing = (apiKey: ApiKey) => {
    setEditingKey({ id: apiKey.id, name: apiKey.name });
    setEditForm({ name: apiKey.name });
    setEditNameError(false);
  };

  const cancelEditing = () => {
    setEditingKey(null);
    setEditForm({ name: '' });
    setEditNameError(false);
  };

  const columns: ColumnsType<ApiKey> = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      width: 300,
      render: (name: string, record: ApiKey) => {
        if (editingKey && editingKey.id === record.id) {
          return (
            <div>
              <div className="mb-1">
                <Input
                  value={editForm.name}
                  onChange={(e) => {
                    setEditNameError(false);
                    setEditForm({ name: e.target.value });
                  }}
                  onPressEnter={handleUpdateApiKey}
                  status={editNameError ? 'error' : undefined}
                  placeholder="Enter API key name"
                  maxLength={100}
                  size="small"
                  style={{ width: 200 }}
                />
              </div>

              <Button
                type="primary"
                size="small"
                onClick={handleUpdateApiKey}
                loading={isUpdating}
                disabled={isUpdating}
                className="border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700"
              >
                Save
              </Button>

              <Button size="small" onClick={cancelEditing} disabled={isUpdating} className="ml-1">
                Cancel
              </Button>
            </div>
          );
        }
        return <span className="font-medium">{name}</span>;
      },
    },
    {
      title: 'Token prefix',
      dataIndex: 'tokenPrefix',
      key: 'tokenPrefix',
      width: 150,
      render: (tokenPrefix: string) => (
        <code className="rounded bg-gray-100 px-2 py-1 font-mono text-sm text-gray-700">
          {tokenPrefix}...
        </code>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: ApiKeyStatus, record: ApiKey) => (
        <div className="flex items-center space-x-2">
          {canManageKeys && (
            <Switch
              checked={status === ApiKeyStatus.ACTIVE}
              onChange={() => handleToggleStatus(record.id, status)}
              loading={processingKeys.has(record.id)}
              disabled={processingKeys.has(record.id)}
              size="small"
              style={{
                backgroundColor: status === ApiKeyStatus.ACTIVE ? '#059669' : undefined,
              }}
            />
          )}
        </div>
      ),
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 200,
      render: (createdAt: Date) => {
        const date = dayjs(createdAt);
        return (
          <div className="text-sm text-gray-600">
            <div>{date.format('MMM D, YYYY')}</div>
            <div className="text-xs text-gray-400">
              {date.format('HH:mm')} ({date.fromNow()})
            </div>
          </div>
        );
      },
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 120,
      render: (_, record: ApiKey) => {
        if (!canManageKeys) return null;

        return (
          <div className="flex items-center space-x-2">
            <Tooltip title="Edit name">
              <Button
                type="text"
                size="small"
                icon={<EditOutlined />}
                onClick={() => startEditing(record)}
                disabled={!!editingKey || processingKeys.has(record.id)}
              />
            </Tooltip>

            <Tooltip title="Delete API key">
              <Popconfirm
                title="Delete API key"
                description={`Are you sure you want to delete "${record.name}"? This action cannot be undone.`}
                onConfirm={() => handleDeleteApiKey(record.id, record.name)}
                okText="Delete"
                cancelText="Cancel"
                okButtonProps={{ danger: true }}
              >
                <Button
                  type="text"
                  size="small"
                  icon={<DeleteOutlined />}
                  danger
                  loading={deletingKeys.has(record.id)}
                  disabled={deletingKeys.has(record.id) || processingKeys.has(record.id)}
                />
              </Popconfirm>
            </Tooltip>
          </div>
        );
      },
    },
  ];

  const isLoading = isLoadingProject || isLoadingApiKeys;

  return (
    <div className="flex grow pl-3">
      <div className="w-full">
        <div
          className="grow overflow-y-auto rounded bg-white p-5 shadow"
          style={{ height: contentHeight }}
        >
          <div className="max-w-[850px]">
            <div className="mb-6 flex items-center justify-between">
              <h1 className="text-2xl font-bold">API keys</h1>
              {canManageKeys && (
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => setIsCreateModalOpen(true)}
                  disabled={isLoading}
                  className="border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700"
                >
                  Create API key
                </Button>
              )}
            </div>

            {/* Warning if API key validation is disabled */}
            {project && !project.isApiKeyRequired && (
              <div className="mb-6 rounded-md border border-yellow-200 bg-yellow-50 p-4">
                <div className="flex items-start">
                  <WarningOutlined className="mt-0.5 mr-2 text-yellow-600" />
                  <div>
                    <div className="font-medium text-yellow-800">API key validation disabled</div>
                    <div className="mt-1 text-sm text-yellow-700">
                      API key validation is currently disabled in project settings. API keys created
                      here won&apos;t be enforced for log ingestion.{' '}
                      <span className="font-medium">
                        Enable &quot;Require API key&quot; in Settings to enforce API key
                        validation.
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {isLoading ? (
              <div className="flex h-64 items-center justify-center">
                <Spin indicator={<LoadingOutlined spin />} size="large" />
              </div>
            ) : (
              <div>
                <div className="mb-4 text-sm text-gray-500">
                  {apiKeys.length === 0
                    ? 'No API keys found'
                    : `${apiKeys.length} API key${apiKeys.length !== 1 ? 's' : ''}`}
                </div>

                <Table
                  columns={columns}
                  dataSource={apiKeys}
                  pagination={false}
                  rowKey="id"
                  size="small"
                  locale={{
                    emptyText: (
                      <div className="py-8 text-center text-gray-500">
                        <div className="mb-2">No API keys created yet</div>
                        {canManageKeys && (
                          <div className="text-sm">
                            Click &quot;Create API key&quot; to get started
                          </div>
                        )}
                      </div>
                    ),
                  }}
                />
              </div>
            )}

            {!canManageKeys && (
              <div className="mt-6 rounded-md bg-yellow-50 p-3">
                <div className="text-sm text-yellow-800">
                  You don&apos;t have permission to manage API keys. Only project owners, project
                  admins, and system administrators can create, edit, or delete API keys.
                </div>
              </div>
            )}

            {/* Create API Key Modal */}
            <Modal
              title="Create new API key"
              open={isCreateModalOpen}
              onOk={handleCreateApiKey}
              onCancel={() => {
                setIsCreateModalOpen(false);
                setCreateForm({ name: '' });
                setCreateNameError(false);
              }}
              confirmLoading={isCreating}
              okText="Create API key"
              cancelText="Cancel"
              okButtonProps={{
                className:
                  'border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700',
              }}
            >
              <div className="py-4">
                <div className="mb-4">
                  <div className="mb-2 font-medium text-gray-900">API key name</div>
                  <Input
                    value={createForm.name}
                    onChange={(e) => {
                      setCreateNameError(false);
                      setCreateForm({ name: e.target.value });
                    }}
                    placeholder="Enter a descriptive name for this API key"
                    maxLength={100}
                    status={createNameError ? 'error' : undefined}
                  />
                  <div className="mt-1 text-xs text-gray-500">
                    Choose a name that helps you identify this key&apos;s purpose (e.g.,
                    &quot;Production&quot;, &quot;Development&quot;)
                  </div>
                </div>
              </div>
            </Modal>

            {/* Token Display Modal */}
            <Modal
              title={
                <div className="flex items-center">
                  <span className="mr-2 text-green-600">✓</span>
                  API key created successfully
                </div>
              }
              open={isTokenModalOpen}
              onCancel={() => {
                setIsTokenModalOpen(false);
                setCreatedApiKey(null);
              }}
              footer={[
                <Button
                  key="close"
                  type="primary"
                  onClick={() => {
                    setIsTokenModalOpen(false);
                    setCreatedApiKey(null);
                  }}
                  className="border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700"
                >
                  I have saved the token
                </Button>,
              ]}
              width={700}
              centered
              maskClosable={false}
              closable={false}
            >
              {createdApiKey && (
                <div className="mt-6">
                  <div className="mb-4">
                    <div className="mb-2 font-medium text-gray-900">API key name:</div>
                    <div className="text-gray-700">{createdApiKey.name}</div>
                  </div>

                  <div className="mb-4">
                    <div className="mb-2 font-medium text-gray-900">Full API token:</div>
                    <div className="rounded-lg border-2 border-emerald-200 bg-emerald-50 p-4">
                      <div className="flex items-center justify-between">
                        <code className="font-mono text-sm break-all text-emerald-800 select-all">
                          {createdApiKey.token}
                        </code>
                        <Button
                          type="primary"
                          icon={<CopyOutlined />}
                          onClick={async () => {
                            if (createdApiKey.token) {
                              const success = await copyToClipboard(createdApiKey.token);
                              if (success) {
                                message.success('API token copied to clipboard!');
                              } else {
                                message.error(
                                  'Failed to copy token to clipboard. Please select and copy the token manually.',
                                );
                              }
                            }
                          }}
                          className="ml-3 border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700"
                        >
                          Copy token
                        </Button>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </Modal>
          </div>
        </div>
      </div>
    </div>
  );
}
