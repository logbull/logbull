import { LoadingOutlined } from '@ant-design/icons';
import { App, Button, Spin, Switch } from 'antd';
import { useEffect, useRef, useState } from 'react';

import { getApplicationServer } from '../../../constants';
import { settingsApi } from '../../../entity/users/api/settingsApi';
import type { UsersSettings } from '../../../entity/users/model/UsersSettings';
import { AuditLogsComponent } from './AuditLogsComponent';

interface Props {
  contentHeight: number;
}

export function SettingsComponent({ contentHeight }: Props) {
  const { message } = App.useApp();
  const [settings, setSettings] = useState<UsersSettings | undefined>(undefined);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);

  // Scroll container ref for audit logs lazy loading
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // Form state to track changes
  const [formSettings, setFormSettings] = useState<UsersSettings>({
    isAllowExternalRegistrations: false,
    isAllowMemberInvitations: false,
    isMemberAllowedToCreateProjects: false,
  });

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    setIsLoading(true);

    try {
      const currentSettings = await settingsApi.getSettings();
      setSettings(currentSettings);
      setFormSettings(currentSettings);
      setHasChanges(false);
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to load settings';
      message.error(errorMessage);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSettingChange = (key: keyof UsersSettings, value: boolean) => {
    const newFormSettings = { ...formSettings, [key]: value };
    setFormSettings(newFormSettings);

    // Check if there are changes from the original settings
    if (settings) {
      const hasAnyChanges = Object.keys(newFormSettings).some(
        (settingKey) =>
          newFormSettings[settingKey as keyof UsersSettings] !==
          settings[settingKey as keyof UsersSettings],
      );
      setHasChanges(hasAnyChanges);
    }
  };

  const handleSave = async () => {
    if (!hasChanges) return;

    setIsSaving(true);
    try {
      const updatedSettings = await settingsApi.updateSettings(formSettings);
      setSettings(updatedSettings);
      setFormSettings(updatedSettings);
      setHasChanges(false);
      message.success('Settings updated successfully');
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to update settings';
      message.error(errorMessage);
    } finally {
      setIsSaving(false);
    }
  };

  const handleReset = () => {
    if (settings) {
      setFormSettings(settings);
      setHasChanges(false);
    }
  };

  return (
    <div className="flex grow pl-3">
      <div className="w-full">
        <div
          ref={scrollContainerRef}
          className="grow overflow-y-auto rounded bg-white p-5 shadow"
          style={{ height: contentHeight }}
        >
          <h1 className="text-2xl font-bold">LogBull Settings</h1>

          <div className="my-6 min-h-[300px]">
            {isLoading ? (
              <div>
                <Spin indicator={<LoadingOutlined spin />} />
              </div>
            ) : (
              <div className="max-w-lg text-sm">
                <div className="space-y-6">
                  {/* External Registrations Setting */}
                  <div className="flex items-start justify-between border-b border-gray-200 pb-4">
                    <div className="flex-1 pr-20">
                      <div className="font-medium text-gray-900">Allow external registrations</div>
                      <div className="mt-1 text-gray-500">
                        When enabled, new users can register accounts in LogBull. If disabled, new
                        users can only register via invitation
                      </div>
                    </div>

                    <div className="ml-4">
                      <Switch
                        checked={formSettings.isAllowExternalRegistrations}
                        onChange={(checked) =>
                          handleSettingChange('isAllowExternalRegistrations', checked)
                        }
                        style={{
                          backgroundColor: formSettings.isAllowExternalRegistrations
                            ? '#059669'
                            : undefined,
                        }}
                      />
                    </div>
                  </div>

                  {/* Member Invitations Setting */}
                  <div className="flex items-start justify-between border-b border-gray-200 pb-4">
                    <div className="flex-1 pr-20">
                      <div className="font-medium text-gray-900">Allow member invitations</div>

                      <div className="mt-1 text-gray-500">
                        When enabled, existing members can invite new users to join LogBull. If not
                        - only admins can invite users.
                      </div>
                    </div>

                    <div className="ml-4">
                      <Switch
                        checked={formSettings.isAllowMemberInvitations}
                        onChange={(checked) =>
                          handleSettingChange('isAllowMemberInvitations', checked)
                        }
                        style={{
                          backgroundColor: formSettings.isAllowMemberInvitations
                            ? '#059669'
                            : undefined,
                        }}
                      />
                    </div>
                  </div>

                  {/* Member Project Creation Setting */}
                  <div className="flex items-start justify-between border-b border-gray-200 pb-4">
                    <div className="flex-1 pr-20">
                      <div className="font-medium text-gray-900">Members can create projects</div>

                      <div className="mt-1 text-gray-500">
                        When enabled, members (non-admin users) can create new projects. If not -
                        only admins can create projects.
                      </div>
                    </div>
                    <div className="ml-4">
                      <Switch
                        checked={formSettings.isMemberAllowedToCreateProjects}
                        onChange={(checked) =>
                          handleSettingChange('isMemberAllowedToCreateProjects', checked)
                        }
                        style={{
                          backgroundColor: formSettings.isMemberAllowedToCreateProjects
                            ? '#059669'
                            : undefined,
                        }}
                      />
                    </div>
                  </div>
                </div>

                {/* Action Buttons */}
                {hasChanges && (
                  <div className="mt-8 flex space-x-2">
                    <Button
                      type="primary"
                      onClick={handleSave}
                      loading={isSaving}
                      disabled={isSaving}
                      className="border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700"
                    >
                      {isSaving ? 'Saving...' : 'Save Changes'}
                    </Button>

                    <Button type="default" onClick={handleReset} disabled={isSaving}>
                      Reset
                    </Button>
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Health-check Information */}
          <div className="my-8 max-w-2xl">
            <h2 className="mb-3 text-xl font-bold">Health-check</h2>

            <div className="group relative">
              <div className="flex items-center rounded-md border border-gray-300 bg-gray-50 px-3 py-2 font-mono text-sm text-gray-700">
                <code
                  className="flex-1 cursor-pointer transition-colors select-all hover:text-emerald-600"
                  onClick={() => {
                    window.open(
                      `${getApplicationServer()}/api/v1/downdetect/is-available`,
                      '_blank',
                    );
                  }}
                  title="Click to open in new tab"
                >
                  {getApplicationServer()}/api/v1/downdetect/is-available
                </code>
                <Button
                  type="text"
                  size="small"
                  className="ml-2 opacity-0 transition-opacity group-hover:opacity-100"
                  onClick={() => {
                    navigator.clipboard.writeText(
                      `${getApplicationServer()}/api/v1/downdetect/is-available`,
                    );
                    message.success('Health-check endpoint copied to clipboard');
                  }}
                >
                  📋
                </Button>
              </div>
              <div className="mt-1 text-xs text-gray-500">
                Use this endpoint to monitor your LogBull system&apos;s availability
              </div>
            </div>
          </div>

          <AuditLogsComponent scrollContainerRef={scrollContainerRef} />
        </div>
      </div>
    </div>
  );
}
