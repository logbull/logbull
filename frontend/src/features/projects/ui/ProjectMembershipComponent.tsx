import {
  DeleteOutlined,
  LoadingOutlined,
  PlusOutlined,
  SwapOutlined,
  UserAddOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { App, Button, Input, Modal, Popconfirm, Select, Spin, Table, Tag, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';

import type {
  AddMemberRequest,
  AddMemberResponse,
  ChangeMemberRoleRequest,
  GetMembersResponse,
  ProjectMemberResponse,
  ProjectResponse,
  TransferOwnershipRequest,
} from '../../../entity/projects';
import { projectMembershipApi } from '../../../entity/projects';
import { AddMemberStatusEnum } from '../../../entity/projects/model/AddMemberStatus';
import type { UserProfile } from '../../../entity/users';
import { ProjectRole } from '../../../entity/users/model/ProjectRole';
import { UserRole } from '../../../entity/users/model/UserRole';
import { StringUtils } from '../../../shared/lib';

interface Props {
  contentHeight: number;
  projectResponse: ProjectResponse;
  user: UserProfile;
}

export function ProjectMembershipComponent({ contentHeight, projectResponse, user }: Props) {
  const { message } = App.useApp();

  // Members state
  const [members, setMembers] = useState<ProjectMemberResponse[]>([]);
  const [isLoadingMembers, setIsLoadingMembers] = useState(true);

  // Add member modal state
  const [isAddMemberModalOpen, setIsAddMemberModalOpen] = useState(false);
  const [addMemberForm, setAddMemberForm] = useState({ email: '', role: ProjectRole.MEMBER });
  const [isAddingMember, setIsAddingMember] = useState(false);
  const [addMemberEmailError, setAddMemberEmailError] = useState(false);

  // Invite dialog state
  const [isInviteDialogOpen, setIsInviteDialogOpen] = useState(false);
  const [invitedEmail, setInvitedEmail] = useState('');

  // Change role state
  const [changingRoleFor, setChangingRoleFor] = useState<string | null>(null);
  const [isChangingRole, setIsChangingRole] = useState(false);

  // Transfer ownership state
  const [isTransferOwnershipModalOpen, setIsTransferOwnershipModalOpen] = useState(false);
  const [transferForm, setTransferForm] = useState({ selectedMemberId: '' });
  const [isTransferringOwnership, setIsTransferringOwnership] = useState(false);
  const [transferMemberError, setTransferMemberError] = useState(false);

  // Processing states
  const [removingMembers, setRemovingMembers] = useState<Set<string>>(new Set());

  // Permissions check
  const canManageMembers =
    user.role === UserRole.ADMIN ||
    projectResponse.userRole === ProjectRole.OWNER ||
    projectResponse.userRole === ProjectRole.ADMIN;

  const canTransferOwnership =
    user.role === UserRole.ADMIN || projectResponse.userRole === ProjectRole.OWNER;

  // Get eligible members for ownership transfer
  const eligibleMembers = members.filter((member) => {
    // Exclude project owners from being eligible
    if (member.role === ProjectRole.OWNER) return false;

    // If this is the current user
    if (member.userId === user.id || member.email === user.email) {
      // Include current user only if they are global admin but NOT project owner
      return user.role === UserRole.ADMIN && projectResponse.userRole !== ProjectRole.OWNER;
    }

    // Include all other non-owner members
    return true;
  });

  const loadMembers = async () => {
    setIsLoadingMembers(true);
    try {
      const response: GetMembersResponse = await projectMembershipApi.getMembers(
        projectResponse.id,
      );
      setMembers(response.members);
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error
          ? StringUtils.capitalizeFirstLetter(error.message)
          : 'Failed to load members';
      message.error(errorMessage);
    } finally {
      setIsLoadingMembers(false);
    }
  };

  const handleAddMember = async () => {
    if (!addMemberForm.email.trim()) {
      setAddMemberEmailError(true);
      message.error('Email is required');
      return;
    }
    setAddMemberEmailError(false);
    setIsAddingMember(true);

    try {
      const request: AddMemberRequest = {
        email: addMemberForm.email.trim(),
        role: addMemberForm.role,
      };
      const response: AddMemberResponse = await projectMembershipApi.addMember(
        projectResponse.id,
        request,
      );

      const emailToRemember = request.email;
      setAddMemberForm({ email: '', role: ProjectRole.MEMBER });
      setIsAddMemberModalOpen(false);

      if (response.status === AddMemberStatusEnum.ADDED) {
        message.success('Member added successfully');
        loadMembers(); // Reload to get updated member list
      } else if (response.status === AddMemberStatusEnum.INVITED) {
        setInvitedEmail(emailToRemember);
        setIsInviteDialogOpen(true);
        loadMembers(); // Reload to get updated member list
      }
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error
          ? StringUtils.capitalizeFirstLetter(error.message)
          : 'Failed to add member';
      message.error(errorMessage);
    } finally {
      setIsAddingMember(false);
    }
  };

  const handleChangeRole = async (userId: string, newRole: ProjectRole) => {
    setChangingRoleFor(userId);
    setIsChangingRole(true);

    try {
      const request: ChangeMemberRoleRequest = { role: newRole };
      await projectMembershipApi.changeMemberRole(projectResponse.id, userId, request);

      // Update local state
      setMembers((prev) =>
        prev.map((member) => (member.userId === userId ? { ...member, role: newRole } : member)),
      );

      message.success('Member role updated successfully');
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error
          ? StringUtils.capitalizeFirstLetter(error.message)
          : 'Failed to change member role';
      message.error(errorMessage);
    } finally {
      setChangingRoleFor(null);
      setIsChangingRole(false);
    }
  };

  const handleRemoveMember = async (userId: string, memberEmail: string) => {
    setRemovingMembers((prev) => new Set(prev).add(userId));

    try {
      await projectMembershipApi.removeMember(projectResponse.id, userId);
      setMembers((prev) => prev.filter((member) => member.userId !== userId));
      message.success(`Member "${memberEmail}" removed successfully`);
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error
          ? StringUtils.capitalizeFirstLetter(error.message)
          : 'Failed to remove member';
      message.error(errorMessage);
    } finally {
      setRemovingMembers((prev) => {
        const newSet = new Set(prev);
        newSet.delete(userId);
        return newSet;
      });
    }
  };

  const handleTransferOwnership = async () => {
    if (!transferForm.selectedMemberId) {
      setTransferMemberError(true);
      message.error('Please select a member to transfer ownership to');
      return;
    }

    // Find the selected member to get their email
    const selectedMember = members.find(
      (member) => member.userId === transferForm.selectedMemberId,
    );
    if (!selectedMember) {
      message.error('Selected member not found');
      return;
    }

    setTransferMemberError(false);
    setIsTransferringOwnership(true);

    try {
      const request: TransferOwnershipRequest = {
        newOwnerEmail: selectedMember.email,
      };
      await projectMembershipApi.transferOwnership(projectResponse.id, request);

      setTransferForm({ selectedMemberId: '' });
      setIsTransferOwnershipModalOpen(false);
      message.success('Ownership transferred successfully');
      loadMembers(); // Reload to get updated member list
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error
          ? StringUtils.capitalizeFirstLetter(error.message)
          : 'Failed to transfer ownership';
      message.error(errorMessage);
    } finally {
      setIsTransferringOwnership(false);
    }
  };

  const getRoleColor = (role: ProjectRole): string => {
    switch (role) {
      case ProjectRole.OWNER:
        return 'red';
      case ProjectRole.ADMIN:
        return 'orange';
      case ProjectRole.MEMBER:
        return 'blue';
      default:
        return 'default';
    }
  };

  const getRoleDisplayText = (role: ProjectRole): string => {
    switch (role) {
      case ProjectRole.OWNER:
        return 'Owner';
      case ProjectRole.ADMIN:
        return 'Admin';
      case ProjectRole.MEMBER:
        return 'Member';
      default:
        return role;
    }
  };

  useEffect(() => {
    loadMembers();
  }, [projectResponse.id]);

  const columns: ColumnsType<ProjectMemberResponse> = [
    {
      title: 'Member',
      key: 'member',
      width: 300,
      render: (_, record: ProjectMemberResponse) => (
        <div className="flex items-center">
          <UserOutlined className="mr-2 text-gray-400" />
          <div>
            <div className="font-medium">{record.email}</div>
            <div className="text-xs text-gray-500">ID: {record.userId}</div>
          </div>
        </div>
      ),
    },
    {
      title: 'Role',
      dataIndex: 'role',
      key: 'role',
      width: 150,
      render: (role: ProjectRole, record: ProjectMemberResponse) => {
        const isCurrentUser = record.userId === user.id || record.email === user.email;

        if (canManageMembers && role !== ProjectRole.OWNER && !isCurrentUser) {
          return (
            <Select
              value={role}
              onChange={(newRole) => handleChangeRole(record.userId, newRole)}
              loading={changingRoleFor === record.userId && isChangingRole}
              disabled={changingRoleFor === record.userId && isChangingRole}
              size="small"
              style={{ width: 100 }}
              options={[
                { label: 'Admin', value: ProjectRole.ADMIN },
                { label: 'Member', value: ProjectRole.MEMBER },
              ]}
            />
          );
        }
        return <Tag color={getRoleColor(role)}>{getRoleDisplayText(role)}</Tag>;
      },
    },
    {
      title: 'Joined',
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
      render: (_, record: ProjectMemberResponse) => {
        const isCurrentUser = record.userId === user.id || record.email === user.email;

        if (!canManageMembers || record.role === ProjectRole.OWNER || isCurrentUser) return null;

        return (
          <div className="flex items-center space-x-2">
            <Tooltip title="Remove member">
              <Popconfirm
                title="Remove member"
                description={`Are you sure you want to remove "${record.email}" from this project?`}
                onConfirm={() => handleRemoveMember(record.userId, record.email)}
                okText="Remove"
                cancelText="Cancel"
                okButtonProps={{ danger: true }}
              >
                <Button
                  type="text"
                  size="small"
                  icon={<DeleteOutlined />}
                  danger
                  loading={removingMembers.has(record.userId)}
                  disabled={removingMembers.has(record.userId)}
                />
              </Popconfirm>
            </Tooltip>
          </div>
        );
      },
    },
  ];

  return (
    <div className="flex grow pl-3">
      <div className="w-full">
        <div
          className="grow overflow-y-auto rounded bg-white p-5 shadow"
          style={{ height: contentHeight }}
        >
          <div className="max-w-[850px]">
            <div className="mb-6 flex items-center justify-between">
              <h1 className="text-2xl font-bold">Project members</h1>
              <div className="flex space-x-2">
                {canTransferOwnership && (
                  <Button
                    icon={<SwapOutlined />}
                    onClick={() => setIsTransferOwnershipModalOpen(true)}
                    disabled={isLoadingMembers || eligibleMembers.length === 0}
                  >
                    Transfer ownership
                  </Button>
                )}
                {canManageMembers && (
                  <Button
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={() => setIsAddMemberModalOpen(true)}
                    disabled={isLoadingMembers}
                    className="border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700"
                  >
                    Add member
                  </Button>
                )}
              </div>
            </div>

            {isLoadingMembers ? (
              <div className="flex h-64 items-center justify-center">
                <Spin indicator={<LoadingOutlined spin />} size="large" />
              </div>
            ) : (
              <div>
                <div className="mb-4 text-sm text-gray-500">
                  {members.length === 0
                    ? 'No members found'
                    : `${members.length} member${members.length !== 1 ? 's' : ''}`}
                </div>

                <Table
                  columns={columns}
                  dataSource={members}
                  pagination={false}
                  rowKey="id"
                  size="small"
                  locale={{
                    emptyText: (
                      <div className="py-8 text-center text-gray-500">
                        <div className="mb-2">No members found</div>
                        {canManageMembers && (
                          <div className="text-sm">Click &quot;Add member&quot; to get started</div>
                        )}
                      </div>
                    ),
                  }}
                />
              </div>
            )}

            {!canManageMembers && (
              <div className="mt-6 rounded-md bg-yellow-50 p-3">
                <div className="text-sm text-yellow-800">
                  You don&apos;t have permission to manage project members. Only project owners,
                  project admins and system administrators can add, edit or remove members.
                </div>
              </div>
            )}

            {/* Add Member Modal */}
            <Modal
              title="Add member"
              open={isAddMemberModalOpen}
              onOk={handleAddMember}
              onCancel={() => {
                setIsAddMemberModalOpen(false);
                setAddMemberForm({ email: '', role: ProjectRole.MEMBER });
                setAddMemberEmailError(false);
              }}
              confirmLoading={isAddingMember}
              okText="Add member"
              cancelText="Cancel"
              okButtonProps={{
                className:
                  'border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700',
              }}
            >
              <div className="py-4">
                <div className="mb-4">
                  <div className="mb-2 font-medium text-gray-900">Email address</div>
                  <Input
                    value={addMemberForm.email}
                    onChange={(e) => {
                      setAddMemberEmailError(false);
                      setAddMemberForm({
                        ...addMemberForm,
                        email: e.target.value.toLowerCase().trim(),
                      });
                    }}
                    placeholder="Enter email address"
                    status={addMemberEmailError ? 'error' : undefined}
                  />
                  <div className="mt-1 text-xs text-gray-500">
                    If the user exists, they will be added directly. Otherwise, an invitation will
                    be sent.
                  </div>
                </div>

                <div className="mb-4">
                  <div className="mb-2 font-medium text-gray-900">Role</div>
                  <Select
                    value={addMemberForm.role}
                    onChange={(role) => setAddMemberForm({ ...addMemberForm, role })}
                    style={{ width: '100%' }}
                    options={[
                      { label: 'Member', value: ProjectRole.MEMBER },
                      { label: 'Admin', value: ProjectRole.ADMIN },
                    ]}
                  />
                </div>
              </div>
            </Modal>

            {/* Invite Dialog */}
            <Modal
              title="User invited"
              open={isInviteDialogOpen}
              onOk={() => setIsInviteDialogOpen(false)}
              onCancel={() => setIsInviteDialogOpen(false)}
              okText="OK"
              cancelButtonProps={{ style: { display: 'none' } }}
              okButtonProps={{
                className:
                  'border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700',
              }}
            >
              <div className="py-4">
                <div className="flex items-center">
                  <UserAddOutlined className="mr-3 text-2xl text-emerald-600" />
                  <div>
                    <div className="font-medium text-gray-900">
                      Invitation sent to {invitedEmail}
                    </div>
                    <div className="mt-1 text-sm text-gray-600">
                      The user is not present in the system yet, but has been invited to the
                      project. After the user signs up via specified email, they will automatically
                      become a member of the project.
                    </div>
                  </div>
                </div>
              </div>
            </Modal>

            {/* Transfer Ownership Modal */}
            <Modal
              title="Transfer project ownership"
              open={isTransferOwnershipModalOpen}
              onOk={handleTransferOwnership}
              onCancel={() => {
                setIsTransferOwnershipModalOpen(false);
                setTransferForm({ selectedMemberId: '' });
                setTransferMemberError(false);
              }}
              confirmLoading={isTransferringOwnership}
              okText="Transfer ownership"
              cancelText="Cancel"
              okButtonProps={{
                danger: true,
                disabled: eligibleMembers.length === 0,
              }}
            >
              <div className="py-4">
                <div className="mb-4 rounded-md bg-yellow-50 p-3">
                  <div className="text-sm text-yellow-800">
                    <strong>Warning:</strong> This action cannot be undone. You will lose ownership
                    of this project and the new owner will have full control.
                  </div>
                </div>

                {eligibleMembers.length === 0 ? (
                  <div className="rounded-md bg-gray-50 p-4 text-center">
                    <div className="text-sm text-gray-600">
                      No members available to transfer ownership to. You need to have at least one
                      other member in the project to transfer ownership.
                    </div>
                  </div>
                ) : (
                  <div className="mb-4">
                    <div className="mb-2 font-medium text-gray-900">Select new owner</div>
                    <Select
                      value={transferForm.selectedMemberId || undefined}
                      onChange={(memberId) => {
                        setTransferMemberError(false);
                        setTransferForm({ selectedMemberId: memberId });
                      }}
                      placeholder="Select a member to transfer ownership to"
                      style={{ width: '100%' }}
                      status={transferMemberError ? 'error' : undefined}
                      options={eligibleMembers.map((member) => ({
                        label: (
                          <div className="flex items-center">
                            <UserOutlined className="mr-2 text-gray-400" />
                            <div>
                              <div className="font-medium">{member.email}</div>
                            </div>
                          </div>
                        ),
                        value: member.userId,
                      }))}
                    />
                    <div className="mt-1 text-xs text-gray-500">
                      The selected member will become the project owner
                    </div>
                  </div>
                )}
              </div>
            </Modal>
          </div>
        </div>
      </div>
    </div>
  );
}
