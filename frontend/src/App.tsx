import { App as AntdApp, ConfigProvider } from 'antd';
import { useEffect, useState } from 'react';
import { BrowserRouter, Route } from 'react-router';
import { Routes } from 'react-router';

import { userApi } from './entity/users';
import { AuthPageComponent } from './pages/AuthPageComponent';
import { MainScreenComponent } from './widgets/main';

function App() {
  const [isAuthorized, setIsAuthorized] = useState(false);

  useEffect(() => {
    const isAuthorized = userApi.isAuthorized();
    setIsAuthorized(isAuthorized);

    userApi.addAuthListener(() => {
      setIsAuthorized(userApi.isAuthorized());
    });
  }, []);

  return (
    <ConfigProvider
      theme={{
        token: {
          colorPrimary: '#009966', // Tailwind emerald-600
        },
      }}
    >
      <AntdApp>
        <BrowserRouter>
          <Routes>
            <Route
              path="/"
              element={!isAuthorized ? <AuthPageComponent /> : <MainScreenComponent />}
            />
          </Routes>
        </BrowserRouter>
      </AntdApp>
    </ConfigProvider>
  );
}

export default App;
