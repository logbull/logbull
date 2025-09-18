import { LoadingOutlined } from '@ant-design/icons';
import { App, Button, Input, InputNumber, Select, Spin, Switch } from 'antd';
import { useEffect, useRef, useState } from 'react';

import { projectApi } from '../../../entity/projects/api/projectApi';
import type { Project } from '../../../entity/projects/model/Project';
import type { ProjectResponse } from '../../../entity/projects/model/ProjectResponse';
import { ProjectRole } from '../../../entity/users/model/ProjectRole';
import type { UserProfile } from '../../../entity/users/model/UserProfile';
import { UserRole } from '../../../entity/users/model/UserRole';
import { ProjectAuditLogsComponent } from './ProjectAuditLogsComponent';

interface Props {
  projectResponse: ProjectResponse;
  user: UserProfile;
  contentHeight: number;
}

export function ProjectSettingsComponent({ projectResponse, user, contentHeight }: Props) {
  const { message, modal } = App.useApp();
  const [project, setProject] = useState<Project | undefined>(undefined);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  // Scroll container ref for audit logs lazy loading
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // Form state to track changes
  const [formProject, setFormProject] = useState<Partial<Project>>({});
  const [nameError, setNameError] = useState(false);
  const [domainErrors, setDomainErrors] = useState<string[]>([]);
  const [ipErrors, setIpErrors] = useState<string[]>([]);

  // Section-specific change tracking
  const [basicInfoChanges, setBasicInfoChanges] = useState(false);
  const [securityPolicyChanges, setSecurityPolicyChanges] = useState(false);
  const [quotasChanges, setQuotasChanges] = useState(false);

  const canEdit =
    user.role === UserRole.ADMIN ||
    projectResponse.userRole === ProjectRole.OWNER ||
    projectResponse.userRole === ProjectRole.ADMIN;

  useEffect(() => {
    loadProject();
  }, [projectResponse.id]);

  // Helper functions to check section-specific changes
  const checkBasicInfoChanges = (newFormProject: Partial<Project>): boolean => {
    if (!project) return false;
    return newFormProject.name !== project.name;
  };

  const checkSecurityPolicyChanges = (newFormProject: Partial<Project>): boolean => {
    if (!project) return false;
    return (
      newFormProject.isApiKeyRequired !== project.isApiKeyRequired ||
      newFormProject.isFilterByDomain !== project.isFilterByDomain ||
      newFormProject.isFilterByIp !== project.isFilterByIp ||
      JSON.stringify(newFormProject.allowedDomains) !== JSON.stringify(project.allowedDomains) ||
      JSON.stringify(newFormProject.allowedIps) !== JSON.stringify(project.allowedIps)
    );
  };

  const checkQuotasChanges = (newFormProject: Partial<Project>): boolean => {
    if (!project) return false;
    return (
      newFormProject.logsPerSecondLimit !== project.logsPerSecondLimit ||
      newFormProject.maxLogsAmount !== project.maxLogsAmount ||
      newFormProject.maxLogsSizeMb !== project.maxLogsSizeMb ||
      newFormProject.maxLogsLifeDays !== project.maxLogsLifeDays ||
      newFormProject.maxLogSizeKb !== project.maxLogSizeKb
    );
  };

  // Validation functions
  const validateDomain = (domain: string): boolean => {
    const domainRegex =
      /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/;
    return domainRegex.test(domain.trim());
  };

  const validateIP = (ip: string): boolean => {
    // IPv4 validation
    const ipv4Regex = /^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}$/;
    // IPv6 validation (simplified)
    const ipv6Regex =
      /^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$/;

    return ipv4Regex.test(ip.trim()) || ipv6Regex.test(ip.trim());
  };

  const validateDomains = (domains: string[]): string[] => {
    const errors: string[] = [];
    domains.forEach((domain, index) => {
      if (!domain.trim()) {
        errors[index] = 'Domain cannot be empty';
      } else if (!validateDomain(domain)) {
        errors[index] = 'Invalid domain format';
      }
    });
    return errors;
  };

  const validateIPs = (ips: string[]): string[] => {
    const errors: string[] = [];
    ips.forEach((ip, index) => {
      if (!ip.trim()) {
        errors[index] = 'IP address cannot be empty';
      } else if (!validateIP(ip)) {
        errors[index] = 'Invalid IP address format';
      }
    });
    return errors;
  };

  const loadProject = async () => {
    setIsLoading(true);

    try {
      const projectData = await projectApi.getProject(projectResponse.id);
      setProject(projectData);
      setFormProject(projectData);
      setNameError(false);
      setDomainErrors([]);
      setIpErrors([]);

      // Reset section-specific change states
      setBasicInfoChanges(false);
      setSecurityPolicyChanges(false);
      setQuotasChanges(false);
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to load project';
      message.error(errorMessage);
    } finally {
      setIsLoading(false);
    }
  };

  const handleFieldChange = <K extends keyof Project>(field: K, value: Project[K]) => {
    const newFormProject = { ...formProject, [field]: value };
    setFormProject(newFormProject);

    // Validate domains and IPs when they change
    if (field === 'allowedDomains' && Array.isArray(value)) {
      const errors = validateDomains(value as string[]);
      setDomainErrors(errors);
    }
    if (field === 'allowedIps' && Array.isArray(value)) {
      const errors = validateIPs(value as string[]);
      setIpErrors(errors);
    }

    // Check section-specific changes
    if (project) {
      setBasicInfoChanges(checkBasicInfoChanges(newFormProject));
      setSecurityPolicyChanges(checkSecurityPolicyChanges(newFormProject));
      setQuotasChanges(checkQuotasChanges(newFormProject));
    }
  };

  // Section-specific save functions
  const saveBasicInfo = async () => {
    if (!basicInfoChanges || !project || !canEdit) return;

    // Validate required fields
    if (!formProject.name?.trim()) {
      setNameError(true);
      message.error('Project name is required');
      return;
    }
    setNameError(false);

    setIsSaving(true);
    try {
      const updateData = {
        ...project,
        name: formProject.name,
      };
      const updatedProject = await projectApi.updateProject(project.id, updateData);
      setProject(updatedProject);
      setFormProject(updatedProject);

      // Only reset basic info changes since that's what we saved
      setBasicInfoChanges(false);

      setNameError(false);
      message.success('Basic information updated successfully');
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to update basic information';
      message.error(errorMessage);
    } finally {
      setIsSaving(false);
    }
  };

  const saveSecurityPolicies = async () => {
    if (!securityPolicyChanges || !project || !canEdit) return;

    // Validate domains and IPs before saving
    if (formProject.isFilterByDomain && formProject.allowedDomains) {
      const domainValidationErrors = validateDomains(formProject.allowedDomains);
      if (domainValidationErrors.some((error) => error)) {
        setDomainErrors(domainValidationErrors);
        message.error('Please fix domain validation errors before saving');
        return;
      }
    }

    if (formProject.isFilterByIp && formProject.allowedIps) {
      const ipValidationErrors = validateIPs(formProject.allowedIps);
      if (ipValidationErrors.some((error) => error)) {
        setIpErrors(ipValidationErrors);
        message.error('Please fix IP address validation errors before saving');
        return;
      }
    }

    setIsSaving(true);
    try {
      const updateData = {
        ...project,
        isApiKeyRequired: formProject.isApiKeyRequired ?? project.isApiKeyRequired,
        isFilterByDomain: formProject.isFilterByDomain ?? project.isFilterByDomain,
        isFilterByIp: formProject.isFilterByIp ?? project.isFilterByIp,
        allowedDomains: formProject.allowedDomains ?? project.allowedDomains,
        allowedIps: formProject.allowedIps ?? project.allowedIps,
      };
      const updatedProject = await projectApi.updateProject(project.id, updateData);
      setProject(updatedProject);
      setFormProject(updatedProject);

      // Only reset security policy changes since that's what we saved
      setSecurityPolicyChanges(false);

      message.success('Security policies updated successfully');
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to update security policies';
      message.error(errorMessage);
    } finally {
      setIsSaving(false);
    }
  };

  const saveQuotas = async () => {
    if (!quotasChanges || !project || !canEdit) return;

    setIsSaving(true);
    try {
      const updateData = {
        ...project,
        logsPerSecondLimit: formProject.logsPerSecondLimit ?? project.logsPerSecondLimit,
        maxLogsAmount: formProject.maxLogsAmount ?? project.maxLogsAmount,
        maxLogsSizeMb: formProject.maxLogsSizeMb ?? project.maxLogsSizeMb,
        maxLogsLifeDays: formProject.maxLogsLifeDays ?? project.maxLogsLifeDays,
        maxLogSizeKb: formProject.maxLogSizeKb ?? project.maxLogSizeKb,
      };
      const updatedProject = await projectApi.updateProject(project.id, updateData);
      setProject(updatedProject);
      setFormProject(updatedProject);

      // Only reset quotas changes since that's what we saved
      setQuotasChanges(false);

      message.success('Rate limiting & quotas updated successfully');
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to update rate limiting & quotas';
      message.error(errorMessage);
    } finally {
      setIsSaving(false);
    }
  };

  const handleDeleteProject = async () => {
    if (!project) {
      message.error('Project not found');
      return;
    }

    if (!canEdit) {
      message.error('You do not have permission to delete this project');
      return;
    }

    modal.confirm({
      title: 'Delete Project',
      content: (
        <div>
          <p>
            Are you sure you want to delete the project <strong>{project.name}</strong>?
          </p>
          <p className="mt-2 text-red-600">
            <strong>This action cannot be undone.</strong> All logs and associated data will be
            permanently removed.
          </p>
        </div>
      ),
      okText: 'Delete Project',
      okType: 'danger',
      cancelText: 'Cancel',
      onOk: async () => {
        setIsDeleting(true);
        try {
          await projectApi.deleteProject(project.id);
          message.success('Project deleted successfully');
          // Redirect to projects list or home page
          window.location.href = '/';
        } catch (error: unknown) {
          const errorMessage = error instanceof Error ? error.message : 'Failed to delete project';
          message.error(errorMessage);
        } finally {
          setIsDeleting(false);
        }
      },
    });
  };

  return (
    <div className="flex grow pl-3">
      <div className="w-full">
        <div
          ref={scrollContainerRef}
          className="grow overflow-y-auto rounded bg-white p-5 shadow"
          style={{ height: contentHeight }}
        >
          <h1 className="mb-6 text-2xl font-bold">Settings</h1>

          {isLoading || !project ? (
            <Spin indicator={<LoadingOutlined spin />} size="large" />
          ) : (
            <>
              {!canEdit && (
                <div className="my-4 rounded-md bg-yellow-50 p-3">
                  <div className="text-sm text-yellow-800">
                    You don&apos;t have permission to modify these settings. Only project owners,
                    project admins and system administrators can change project settings.
                  </div>
                </div>
              )}

              <div className="space-y-6 text-sm">
                <div className="max-w-2xl border-b border-gray-200 pb-6">
                  <div className="max-w-md">
                    <div className="mb-1 font-medium text-gray-900">Project name</div>
                    <Input
                      value={formProject.name || ''}
                      onChange={(e) => {
                        setNameError(false);
                        handleFieldChange('name', e.target.value);
                      }}
                      disabled={!canEdit}
                      placeholder="Enter project name"
                      maxLength={100}
                      status={nameError ? 'error' : undefined}
                    />
                  </div>

                  {/* Basic Info Save Button */}
                  {basicInfoChanges && canEdit && (
                    <div className="mt-4 flex space-x-2">
                      <Button
                        type="primary"
                        onClick={saveBasicInfo}
                        loading={isSaving}
                        disabled={isSaving}
                        className="border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700"
                      >
                        {isSaving ? 'Saving...' : 'Save Changes'}
                      </Button>

                      <Button
                        type="default"
                        onClick={() => {
                          if (project) {
                            const updatedForm = { ...formProject, name: project.name };
                            setFormProject(updatedForm);
                            setBasicInfoChanges(false);
                            setNameError(false);
                          }
                        }}
                        disabled={isSaving}
                      >
                        Reset
                      </Button>
                    </div>
                  )}
                </div>

                {/* Security Policies */}
                <div className="max-w-2xl border-b border-gray-200 pb-6">
                  <h2 className="mb-4 text-xl font-bold text-gray-900">Security policies</h2>

                  <div className="space-y-4">
                    <div className="flex items-start justify-between">
                      <div className="flex-1 pr-20">
                        <div className="font-medium text-gray-900">Require API key</div>
                        <div className="mt-1 text-gray-500">
                          When enabled, all log ingestion requests must include a valid API key
                        </div>
                      </div>
                      <div className="ml-4">
                        <Switch
                          checked={formProject.isApiKeyRequired ?? false}
                          onChange={(checked) => handleFieldChange('isApiKeyRequired', checked)}
                          disabled={!canEdit}
                          style={{
                            backgroundColor: formProject.isApiKeyRequired ? '#059669' : undefined,
                          }}
                        />
                      </div>
                    </div>

                    <div className="space-y-3">
                      <div className="flex items-start justify-between">
                        <div className="flex-1 pr-20">
                          <div className="font-medium text-gray-900">Filter by domain</div>
                          <div className="mt-1 text-gray-500">
                            When enabled, only requests from allowed domains will be accepted
                          </div>
                        </div>
                        <div className="ml-4">
                          <Switch
                            checked={formProject.isFilterByDomain ?? false}
                            onChange={(checked) => {
                              const newFormProject = {
                                ...formProject,
                                isFilterByDomain: checked,
                                ...(checked ? {} : { allowedDomains: [] }),
                              };
                              setFormProject(newFormProject);

                              if (!checked) {
                                setDomainErrors([]);
                              }

                              // Check section-specific changes
                              if (project) {
                                setBasicInfoChanges(checkBasicInfoChanges(newFormProject));
                                setSecurityPolicyChanges(
                                  checkSecurityPolicyChanges(newFormProject),
                                );
                                setQuotasChanges(checkQuotasChanges(newFormProject));
                              }
                            }}
                            disabled={!canEdit}
                            style={{
                              backgroundColor: formProject.isFilterByDomain ? '#059669' : undefined,
                            }}
                          />
                        </div>
                      </div>

                      {formProject.isFilterByDomain && (
                        <div className="ml-0">
                          <div className="mb-2 text-sm font-medium text-gray-700">
                            Allowed domains
                          </div>
                          <Select
                            mode="tags"
                            value={formProject.allowedDomains || []}
                            onChange={(value) => handleFieldChange('allowedDomains', value)}
                            disabled={!canEdit}
                            placeholder="Enter domains (e.g., example.com, subdomain.example.com)"
                            className="w-full"
                            style={
                              {
                                '--ant-color-primary': '#059669',
                                '--ant-color-primary-hover': '#047857',
                              } as React.CSSProperties
                            }
                            status={domainErrors.some((error) => error) ? 'error' : undefined}
                            tokenSeparators={[',', ' ']}
                          />
                          {domainErrors.length > 0 && domainErrors.some((error) => error) && (
                            <div className="mt-1 text-xs text-red-600">
                              {domainErrors.map((error, index) =>
                                error ? (
                                  <div key={index}>
                                    Domain {index + 1}: {error}
                                  </div>
                                ) : null,
                              )}
                            </div>
                          )}
                          <div className="mt-1 text-xs text-gray-500">
                            Press Enter or comma to add multiple domains. Only requests from these
                            domains will be accepted.
                          </div>
                        </div>
                      )}
                    </div>

                    <div className="space-y-3">
                      <div className="flex items-start justify-between">
                        <div className="flex-1 pr-20">
                          <div className="font-medium text-gray-900">Filter by IP address</div>
                          <div className="mt-1 text-gray-500">
                            When enabled, only requests from allowed IP addresses will be accepted
                          </div>
                        </div>
                        <div className="ml-4">
                          <Switch
                            checked={formProject.isFilterByIp ?? false}
                            onChange={(checked) => {
                              const newFormProject = {
                                ...formProject,
                                isFilterByIp: checked,
                                ...(checked ? {} : { allowedIps: [] }),
                              };
                              setFormProject(newFormProject);

                              if (!checked) {
                                setIpErrors([]);
                              }

                              // Check section-specific changes
                              if (project) {
                                setBasicInfoChanges(checkBasicInfoChanges(newFormProject));
                                setSecurityPolicyChanges(
                                  checkSecurityPolicyChanges(newFormProject),
                                );
                                setQuotasChanges(checkQuotasChanges(newFormProject));
                              }
                            }}
                            disabled={!canEdit}
                            style={{
                              backgroundColor: formProject.isFilterByIp ? '#059669' : undefined,
                            }}
                          />
                        </div>
                      </div>

                      {formProject.isFilterByIp && (
                        <div className="ml-0">
                          <div className="mb-2 text-sm font-medium text-gray-700">
                            Allowed IP addresses
                          </div>
                          <Select
                            mode="tags"
                            value={formProject.allowedIps || []}
                            onChange={(value) => handleFieldChange('allowedIps', value)}
                            disabled={!canEdit}
                            placeholder="Enter IP addresses (e.g., 192.168.1.1, 10.0.0.0/8)"
                            className="w-full"
                            style={
                              {
                                '--ant-color-primary': '#059669',
                                '--ant-color-primary-hover': '#047857',
                              } as React.CSSProperties
                            }
                            status={ipErrors.some((error) => error) ? 'error' : undefined}
                            tokenSeparators={[',', ' ']}
                          />
                          {ipErrors.length > 0 && ipErrors.some((error) => error) && (
                            <div className="mt-1 text-xs text-red-600">
                              {ipErrors.map((error, index) =>
                                error ? (
                                  <div key={index}>
                                    IP {index + 1}: {error}
                                  </div>
                                ) : null,
                              )}
                            </div>
                          )}
                          <div className="mt-1 text-xs text-gray-500">
                            Press Enter or comma to add multiple IP addresses. Supports both IPv4
                            and IPv6 formats.
                          </div>
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Security Policies Save Button */}
                  {securityPolicyChanges && canEdit && (
                    <div className="mt-4 flex space-x-2">
                      <Button
                        type="primary"
                        onClick={saveSecurityPolicies}
                        loading={isSaving}
                        disabled={isSaving}
                        className="border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700"
                      >
                        {isSaving ? 'Saving...' : 'Save Changes'}
                      </Button>
                      <Button
                        type="default"
                        onClick={() => {
                          if (project) {
                            const updatedForm = {
                              ...formProject,
                              isApiKeyRequired: project.isApiKeyRequired,
                              isFilterByDomain: project.isFilterByDomain,
                              isFilterByIp: project.isFilterByIp,
                              allowedDomains: project.allowedDomains,
                              allowedIps: project.allowedIps,
                            };
                            setFormProject(updatedForm);
                            setSecurityPolicyChanges(false);
                            setDomainErrors([]);
                            setIpErrors([]);
                          }
                        }}
                        disabled={isSaving}
                      >
                        Reset
                      </Button>
                    </div>
                  )}
                </div>

                {/* Rate Limiting & Quotas */}
                <div className="max-w-2xl border-b border-gray-200 pb-6">
                  <h2 className="mb-4 text-xl font-bold text-gray-900">Rate limiting & quotas</h2>

                  <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <div>
                      <div className="mb-1 font-medium text-gray-900">Logs per second limit</div>
                      <InputNumber
                        value={formProject.logsPerSecondLimit}
                        onChange={(value) => handleFieldChange('logsPerSecondLimit', value || 0)}
                        disabled={!canEdit}
                        formatter={(value) => `${value}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
                        parser={(value) => value?.replace(/\$\s?|(,*)/g, '') as unknown as number}
                        min={0}
                        max={100000}
                        style={{ width: '150px' }}
                      />
                      <div className="mt-1 text-xs text-gray-500">
                        Maximum logs that can be ingested per second
                      </div>
                    </div>

                    <div>
                      <div className="mb-1 font-medium text-gray-900">Maximum log size (KB)</div>
                      <InputNumber
                        value={formProject.maxLogSizeKb}
                        onChange={(value) => handleFieldChange('maxLogSizeKb', value || 0)}
                        disabled={!canEdit}
                        formatter={(value) => `${value}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
                        parser={(value) => value?.replace(/\$\s?|(,*)/g, '') as unknown as number}
                        min={1}
                        max={1024}
                        style={{ width: '150px' }}
                      />
                      <div className="mt-1 text-xs text-gray-500">
                        Maximum size allowed for a single log entry
                      </div>
                    </div>

                    <div>
                      <div className="mb-1 font-medium text-gray-900">Maximum logs amount</div>
                      <InputNumber
                        value={formProject.maxLogsAmount}
                        onChange={(value) => handleFieldChange('maxLogsAmount', value || 0)}
                        disabled={!canEdit}
                        min={0}
                        max={1000000000000000}
                        formatter={(value) => `${value}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
                        parser={(value) => value?.replace(/\$\s?|(,*)/g, '') as unknown as number}
                        style={{ width: '150px' }}
                      />
                      <div className="mt-1 text-xs text-gray-500">
                        Maximum total number of logs that can be stored
                      </div>
                    </div>

                    <div>
                      <div className="mb-1 font-medium text-gray-900">
                        Maximum storage size (MB)
                      </div>
                      <InputNumber
                        value={formProject.maxLogsSizeMb}
                        onChange={(value) => handleFieldChange('maxLogsSizeMb', value || 0)}
                        formatter={(value) => `${value}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
                        parser={(value) => value?.replace(/\$\s?|(,*)/g, '') as unknown as number}
                        disabled={!canEdit}
                        min={0}
                        max={1000000000000000}
                        style={{ width: '150px' }}
                      />
                      <div className="mt-1 text-xs text-gray-500">
                        Maximum total storage size for all logs
                      </div>
                    </div>

                    <div>
                      <div className="mb-1 font-medium text-gray-900">Log retention (days)</div>
                      <InputNumber
                        value={formProject.maxLogsLifeDays}
                        onChange={(value) => handleFieldChange('maxLogsLifeDays', value || 0)}
                        formatter={(value) => `${value}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
                        parser={(value) => value?.replace(/\$\s?|(,*)/g, '') as unknown as number}
                        disabled={!canEdit}
                        min={1}
                        max={3650}
                        style={{ width: '150px' }}
                      />
                      <div className="mt-1 text-xs text-gray-500">
                        How long logs should be kept before automatic deletion
                      </div>
                    </div>
                  </div>

                  {/* Rate Limiting & Quotas Save Button */}
                  {quotasChanges && canEdit && (
                    <div className="mt-4 flex space-x-2">
                      <Button
                        type="primary"
                        onClick={saveQuotas}
                        loading={isSaving}
                        disabled={isSaving}
                        className="border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700"
                      >
                        {isSaving ? 'Saving...' : 'Save Changes'}
                      </Button>
                      <Button
                        type="default"
                        onClick={() => {
                          if (project) {
                            const updatedForm = {
                              ...formProject,
                              logsPerSecondLimit: project.logsPerSecondLimit,
                              maxLogsAmount: project.maxLogsAmount,
                              maxLogsSizeMb: project.maxLogsSizeMb,
                              maxLogsLifeDays: project.maxLogsLifeDays,
                              maxLogSizeKb: project.maxLogSizeKb,
                            };
                            setFormProject(updatedForm);
                            setQuotasChanges(false);
                          }
                        }}
                        disabled={isSaving}
                      >
                        Reset
                      </Button>
                    </div>
                  )}
                </div>

                {/* Project Deletion */}
                <div className="max-w-2xl border-b border-gray-200 pb-6">
                  <div className="rounded-lg border border-red-200 bg-red-50 p-4">
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="font-medium text-red-900">Delete this project</div>
                        <div className="mt-1 text-sm text-red-700">
                          Once you delete a project, there is no going back. All logs and data
                          associated with this project will be permanently removed.
                        </div>
                      </div>

                      <div className="ml-4">
                        <Button
                          type="primary"
                          danger
                          onClick={handleDeleteProject}
                          disabled={!canEdit || isDeleting || isSaving}
                          loading={isDeleting}
                          className="bg-red-600 hover:bg-red-700"
                        >
                          {isDeleting ? 'Deleting...' : 'Delete project'}
                        </Button>
                      </div>
                    </div>
                  </div>
                </div>

                <ProjectAuditLogsComponent
                  projectId={project.id}
                  scrollContainerRef={scrollContainerRef}
                />
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
