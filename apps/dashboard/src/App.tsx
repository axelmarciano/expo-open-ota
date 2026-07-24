import { Layout } from '@/containers/Layout';
import { Navigate, Route, Routes, useNavigate } from 'react-router';
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
import { Users } from '@/pages/Users';
import { Account } from '@/pages/Account';
import { License } from '@/ee/pages/License';
import { Sso } from '@/ee/pages/Sso';
import { Roles } from '@/ee/pages/Roles';
import { AuditLog } from '@/ee/pages/AuditLog';
import { SettingsProvider } from '@/lib/SettingsContext';
import { CurrentUserProvider } from '@/lib/CurrentUserContext';
import { PermissionsProvider } from '@/ee/lib/PermissionsContext';
import { RequiresApp } from '@/components/RequiresApp';
import { Branches } from '@/pages/Branches';
import { useSettings } from '@/lib/SettingsContext';

function withLayout(children: ReactNode) {
  return <Layout>{children}</Layout>;
}

// App-scoped pages have nothing to render without a visible app: RequiresApp
// swaps them for the "No app available" empty state.
function withApp(children: ReactNode) {
  return <RequiresApp>{children}</RequiresApp>;
}

const DashboardHome = () => {
  const { CONTROL_PLANE_ENABLED } = useSettings();
  return <Navigate to={CONTROL_PLANE_ENABLED ? '/updates' : '/branches'} replace />;
};

// The update feed is control-plane only (a stateless server has no update
// store to feed it): deep links degrade to the branches view instead of a
// page whose query can only fail.
const RequiresControlPlane = ({ children }: { children: ReactNode }) => {
  const { CONTROL_PLANE_ENABLED } = useSettings();
  return CONTROL_PLANE_ENABLED ? <>{children}</> : <Navigate to="/branches" replace />;
};

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
                <CurrentUserProvider>
                  <PermissionsProvider>
                    <SelectedAppProvider>
                      <Routes>
                        <Route path="/" element={withLayout(withApp(<DashboardHome />))} />
                        <Route path="/settings" element={withLayout(<Settings />)} />
                        <Route path="/channels" element={withLayout(withApp(<Channels />))} />
                        <Route
                          path="/channels/:channelName"
                          element={withLayout(withApp(<Channels />))}
                        />
                        <Route path="/branches" element={withLayout(withApp(<Branches />))} />
                        <Route
                          path="/branches/:branchName"
                          element={withLayout(withApp(<Branches />))}
                        />
                        <Route
                          path="/branches/:branchName/runtime-versions/:runtimeVersion"
                          element={withLayout(withApp(<Branches />))}
                        />
                        <Route
                          path="/updates"
                          element={withLayout(
                            withApp(
                              <RequiresControlPlane>
                                <Updates />
                              </RequiresControlPlane>
                            )
                          )}
                        />
                        <Route path="/app-info" element={withLayout(withApp(<AppInfo />))} />
                        <Route path="/tokens" element={withLayout(withApp(<ApiTokens />))} />
                        <Route path="/users" element={withLayout(<Users />)} />
                        <Route path="/roles" element={withLayout(<Roles />)} />
                        <Route path="/audit-logs" element={withLayout(<AuditLog />)} />
                        <Route path="/sso" element={withLayout(<Sso />)} />
                        <Route path="/license" element={withLayout(<License />)} />
                        <Route path="/account" element={withLayout(<Account />)} />
                        <Route path="/logout" element={withLayout(<Logout />)} />
                      </Routes>
                    </SelectedAppProvider>
                  </PermissionsProvider>
                </CurrentUserProvider>
              </SettingsProvider>
            ) : null
          }
        />
      </Routes>
    </>
  );
};
