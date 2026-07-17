import { Layout } from '@/containers/Layout';
import { Route, Routes, useNavigate } from 'react-router';
import { isAuthenticated } from '@/lib/auth.ts';
import { useEffect, ReactNode } from 'react';
import { Login } from '@/pages/Login';
import { Toaster } from '@/components/ui/toaster.tsx';
import { Updates } from '@/pages/Updates';
import { Settings } from '@/pages/Settings';
import { Logout } from '@/pages/Logout';
import { Channels } from '@/pages/Channels';
import { SelectedAppProvider } from '@/lib/SelectedAppContext';
import { AppInfo } from '@/pages/AppInfo';
import { ApiTokens } from '@/pages/ApiTokens';
import { SettingsProvider } from '@/lib/SettingsContext';

function withLayout(children: ReactNode) {
  return <Layout>{children}</Layout>;
}

export const App = () => {
  const isLoggedIn = isAuthenticated();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isLoggedIn) {
      navigate('/login');
    }
  }, [isLoggedIn, navigate]);

  return (
    <>
      <Toaster />
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route
          path="*"
          element={
            isLoggedIn ? (
              <SettingsProvider>
                <SelectedAppProvider>
                  <Routes>
                    <Route path="/" element={withLayout(<Updates />)} />
                    <Route path="/settings" element={withLayout(<Settings />)} />
                    <Route path="/channels" element={withLayout(<Channels />)} />
                    <Route path="/app-info" element={withLayout(<AppInfo />)} />
                    <Route path="/tokens" element={withLayout(<ApiTokens />)} />
                    <Route path="/logout" element={withLayout(<Logout />)} />
                  </Routes>
                </SelectedAppProvider>
              </SettingsProvider>
            ) : null
          }
        />
      </Routes>
    </>
  );
};
