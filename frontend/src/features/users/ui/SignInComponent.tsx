import { EyeInvisibleOutlined, EyeTwoTone } from '@ant-design/icons';
import { Button, Input } from 'antd';
import { type JSX, useState } from 'react';

import { userApi } from '../../../entity/users';
import { StringUtils } from '../../../shared/lib';
import { FormValidator } from '../../../shared/lib/FormValidator';

interface SignInComponentProps {
  onSwitchToSignUp?: () => void;
}

export function SignInComponent({ onSwitchToSignUp }: SignInComponentProps): JSX.Element {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [passwordVisible, setPasswordVisible] = useState(false);

  const [isLoading, setLoading] = useState(false);

  const [isEmailError, setEmailError] = useState(false);
  const [passwordError, setPasswordError] = useState(false);

  const [signInError, setSignInError] = useState('');

  const validateFieldsForSignIn = (): boolean => {
    if (!email) {
      setEmailError(true);
      return false;
    }

    if (!FormValidator.isValidEmail(email) && email !== 'admin') {
      setEmailError(true);
      return false;
    }

    if (!password) {
      setPasswordError(true);
      return false;
    }
    setPasswordError(false);

    return true;
  };

  const onSignIn = async () => {
    setSignInError('');

    if (validateFieldsForSignIn()) {
      setLoading(true);

      try {
        await userApi.signIn({
          email,
          password,
        });
      } catch (e) {
        setSignInError(StringUtils.capitalizeFirstLetter((e as Error).message));
      }

      setLoading(false);
    }
  };

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-5 text-center text-2xl font-bold">Sign in</div>

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

      <div className="mt-3" />

      <Button
        disabled={isLoading}
        loading={isLoading}
        className="w-full"
        onClick={() => {
          onSignIn();
        }}
        type="primary"
      >
        Sign in
      </Button>

      {signInError && (
        <div className="mt-3 flex justify-center text-center text-sm text-red-600">
          {signInError}
        </div>
      )}

      {onSwitchToSignUp && (
        <div className="mt-4 text-center text-sm text-gray-600">
          Don&apos;t have an account?{' '}
          <button
            type="button"
            onClick={onSwitchToSignUp}
            className="cursor-pointer font-medium text-emerald-600 hover:text-emerald-700"
          >
            Sign up
          </button>
        </div>
      )}
    </div>
  );
}
