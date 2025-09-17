import { EyeInvisibleOutlined, EyeTwoTone, LoadingOutlined } from '@ant-design/icons';
import { App, Button, Input, Spin } from 'antd';
import { useEffect, useState } from 'react';

import { userApi } from '../../../entity/users/api/userApi';
import type { ChangePasswordRequest } from '../../../entity/users/model/ChangePasswordRequest';
import type { SignInRequest } from '../../../entity/users/model/SignInRequest';
import type { UserProfile } from '../../../entity/users/model/UserProfile';
import { UserRole } from '../../../entity/users/model/UserRole';

interface Props {
  contentHeight: number;
}

const getRoleDisplayText = (role: UserRole): string => {
  switch (role) {
    case UserRole.ADMIN:
      return 'Admin';
    case UserRole.MEMBER:
      return 'Member';
    default:
      return role;
  }
};

export function ProfileComponent({ contentHeight }: Props) {
  const { message } = App.useApp();
  const [user, setUser] = useState<UserProfile | undefined>(undefined);
  const [isChangingPassword, setIsChangingPassword] = useState(false);

  // Password change form state
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [newPasswordVisible, setNewPasswordVisible] = useState(false);
  const [confirmPasswordVisible, setConfirmPasswordVisible] = useState(false);

  // Error states
  const [newPasswordError, setNewPasswordError] = useState(false);
  const [confirmPasswordError, setConfirmPasswordError] = useState(false);

  useEffect(() => {
    userApi
      .getCurrentUser()
      .then((user) => {
        setUser(user);
      })
      .catch((error) => {
        message.error(error.message);
      });
  }, []);

  const validatePasswordFields = (): boolean => {
    let isValid = true;

    if (!newPassword) {
      setNewPasswordError(true);
      isValid = false;
    } else if (newPassword.length < 6) {
      setNewPasswordError(true);
      message.error('Password must be at least 6 characters long');
      isValid = false;
    } else {
      setNewPasswordError(false);
    }

    if (!confirmPassword) {
      setConfirmPasswordError(true);
      isValid = false;
    } else if (newPassword !== confirmPassword) {
      setConfirmPasswordError(true);
      message.error('New passwords do not match');
      isValid = false;
    } else {
      setConfirmPasswordError(false);
    }

    return isValid;
  };

  const handlePasswordChange = async () => {
    if (!validatePasswordFields()) {
      return;
    }

    setIsChangingPassword(true);

    try {
      const request: ChangePasswordRequest = {
        newPassword,
      };

      await userApi.changePassword(request);

      // Reset form fields
      setNewPassword('');
      setConfirmPassword('');

      // Sign in again with new password
      if (user?.email) {
        try {
          const signInRequest: SignInRequest = {
            email: user.email,
            password: newPassword,
          };
          await userApi.signIn(signInRequest);
          message.success('Successfully signed in with new password');
        } catch (signInError: unknown) {
          const errorMessage =
            signInError instanceof Error
              ? signInError.message
              : 'Failed to sign in with new password';
          message.error(errorMessage);
          // If sign in fails, logout and redirect to login page
          userApi.logout();
          userApi.notifyAuthListeners();
          window.location.reload();
        }
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to change password';
      message.error(errorMessage);
    } finally {
      setIsChangingPassword(false);
    }
  };

  const handleLogout = () => {
    userApi.logout();
    window.location.reload();
  };

  return (
    <div className="flex grow pl-3">
      <div className="w-full">
        <div
          className="grow overflow-y-auto rounded bg-white p-5 shadow"
          style={{ height: contentHeight }}
        >
          <h1 className="text-2xl font-bold">Profile</h1>

          <div className="mt-5">
            {user ? (
              <>
                <div className="mb-6 text-sm">
                  <div className="flex">
                    <div className="w-[60px] font-medium">ID</div> {user.id}
                  </div>
                  <div className="mt-1 flex">
                    <div className="w-[60px] font-medium">Email</div> {user.email}
                  </div>
                  <div className="mt-1 flex items-center">
                    <div className="w-[60px] font-medium">Role</div>
                    <span className="inline-flex items-center rounded-full bg-emerald-100 px-2.5 py-0.5 text-xs font-medium text-emerald-800">
                      {getRoleDisplayText(user.role)}
                    </span>
                  </div>
                </div>

                <div className="mb-8">
                  <Button type="default" onClick={handleLogout} danger>
                    Logout
                  </Button>
                </div>

                <div className="max-w-xs pt-6">
                  <h3 className="mb-4 text-lg font-semibold">Change Password</h3>

                  <div className="max-w-sm">
                    <div className="my-1 text-xs font-semibold">New Password</div>
                    <Input.Password
                      placeholder="Enter new password"
                      value={newPassword}
                      onChange={(e) => {
                        setNewPasswordError(false);
                        setNewPassword(e.currentTarget.value);
                      }}
                      status={newPasswordError ? 'error' : undefined}
                      iconRender={(visible) =>
                        visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />
                      }
                      visibilityToggle={{
                        visible: newPasswordVisible,
                        onVisibleChange: setNewPasswordVisible,
                      }}
                    />

                    <div className="my-1 text-xs font-semibold">Confirm New Password</div>
                    <Input.Password
                      placeholder="Confirm new password"
                      value={confirmPassword}
                      onChange={(e) => {
                        setConfirmPasswordError(false);
                        setConfirmPassword(e.currentTarget.value);
                      }}
                      status={confirmPasswordError ? 'error' : undefined}
                      iconRender={(visible) =>
                        visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />
                      }
                      visibilityToggle={{
                        visible: confirmPasswordVisible,
                        onVisibleChange: setConfirmPasswordVisible,
                      }}
                    />

                    <div className="mt-3" />

                    <Button
                      type="primary"
                      onClick={handlePasswordChange}
                      loading={isChangingPassword}
                      disabled={isChangingPassword}
                      className="border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700"
                    >
                      {isChangingPassword ? 'Changing Password...' : 'Change Password'}
                    </Button>
                  </div>
                </div>
              </>
            ) : (
              <div>
                <Spin indicator={<LoadingOutlined spin />} />
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
