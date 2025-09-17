import { EyeInvisibleOutlined, EyeTwoTone } from '@ant-design/icons';
import { App, Button, Input } from 'antd';
import { type JSX, useState } from 'react';

import { userApi } from '../../../entity/users';
import { StringUtils } from '../../../shared/lib';
import { FormValidator } from '../../../shared/lib/FormValidator';

interface SignUpComponentProps {
  onSwitchToSignIn?: () => void;
}

export function SignUpComponent({ onSwitchToSignIn }: SignUpComponentProps): JSX.Element {
  const { message } = App.useApp();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [passwordVisible, setPasswordVisible] = useState(false);
  const [confirmPassword, setConfirmPassword] = useState('');
  const [confirmPasswordVisible, setConfirmPasswordVisible] = useState(false);

  const [isLoading, setLoading] = useState(false);

  const [isEmailError, setEmailError] = useState(false);
  const [passwordError, setPasswordError] = useState(false);
  const [confirmPasswordError, setConfirmPasswordError] = useState(false);

  const [signUpError, setSignUpError] = useState('');

  const validateFieldsForSignUp = (): boolean => {
    if (!email) {
      setEmailError(true);
      return false;
    }

    if (!FormValidator.isValidEmail(email)) {
      setEmailError(true);
      return false;
    }

    if (!password) {
      setPasswordError(true);
      return false;
    }

    if (password.length < 8) {
      setPasswordError(true);
      message.error('Password must be at least 8 characters long');
      return false;
    }
    setPasswordError(false);

    if (!confirmPassword) {
      setConfirmPasswordError(true);
      return false;
    }
    if (password !== confirmPassword) {
      setConfirmPasswordError(true);
      return false;
    }
    setConfirmPasswordError(false);

    return true;
  };

  const onSignUp = async () => {
    setSignUpError('');

    if (validateFieldsForSignUp()) {
      setLoading(true);

      try {
        await userApi.signUp({
          email,
          password,
        });
        await userApi.signIn({ email, password });
      } catch (e) {
        setSignUpError(StringUtils.capitalizeFirstLetter((e as Error).message));
      }
    }

    setLoading(false);
  };

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-5 text-center text-2xl font-bold">Sign up</div>

      <div className="my-1 text-xs font-semibold">Your email</div>
      <Input
        placeholder="your@email.com"
        value={email}
        onChange={(e) => {
          setEmailError(false);
          setEmail(e.currentTarget.value.trim().toLowerCase());
        }}
        status={isEmailError ? 'error' : undefined}
        type="email"
      />

      <div className="my-1 text-xs font-semibold">Password</div>
      <Input.Password
        placeholder="********"
        value={password}
        onChange={(e) => {
          setPasswordError(false);
          setPassword(e.currentTarget.value);
        }}
        status={passwordError ? 'error' : undefined}
        iconRender={(visible) => (visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />)}
        visibilityToggle={{ visible: passwordVisible, onVisibleChange: setPasswordVisible }}
      />

      <div className="my-1 text-xs font-semibold">Confirm password</div>
      <Input.Password
        placeholder="********"
        value={confirmPassword}
        status={confirmPasswordError ? 'error' : undefined}
        onChange={(e) => {
          setConfirmPasswordError(false);
          setConfirmPassword(e.currentTarget.value);
        }}
        iconRender={(visible) => (visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />)}
        visibilityToggle={{
          visible: confirmPasswordVisible,
          onVisibleChange: setConfirmPasswordVisible,
        }}
      />

      <div className="mt-3" />

      <Button
        disabled={isLoading}
        loading={isLoading}
        className="w-full"
        onClick={() => {
          onSignUp();
        }}
        type="primary"
      >
        Sign up
      </Button>

      {signUpError && (
        <div className="mt-3 flex justify-center text-center text-sm text-red-600">
          {signUpError}
        </div>
      )}

      {onSwitchToSignIn && (
        <div className="mt-4 text-center text-sm text-gray-600">
          Already have an account?{' '}
          <button
            type="button"
            onClick={onSwitchToSignIn}
            className="cursor-pointer font-medium text-emerald-600 hover:text-emerald-700"
          >
            Sign in
          </button>
        </div>
      )}
    </div>
  );
}
