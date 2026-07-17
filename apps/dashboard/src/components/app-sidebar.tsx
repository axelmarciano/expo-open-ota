import { useState } from 'react';
import { Link, useLocation } from 'react-router';
import {
  Box,
  CircleUser,
  HardDriveDownload,
  Info,
  KeyRound,
  LogOut,
  Plus,
  Radio,
  Settings,
  Users,
} from 'lucide-react';
import clsx from 'clsx';
import { Combobox } from '@/components/Combobox';
import { useSelectedApp } from '@/lib/SelectedAppContext';
import { CreateAppModal } from '@/components/app-creation-modal';
import { useSettings } from '@/lib/SettingsContext';
import { useCurrentUser } from '@/lib/CurrentUserContext';

const NavLink = ({
  to,
  icon: Icon,
  children,
}: {
  to: string;
  icon: typeof Box;
  children: React.ReactNode;
}) => {
  const { pathname } = useLocation();
  const isActive = pathname === to;
  return (
    <Link
      to={to}
      onClick={e => {
        if (isActive) e.preventDefault();
      }}
      className={clsx(
        'flex items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors',
        isActive
          ? 'bg-secondary font-medium text-foreground'
          : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
      )}>
      <Icon className="h-4 w-4" strokeWidth={1.75} />
      <span>{children}</span>
    </Link>
  );
};

const SectionLabel = ({ children }: { children: React.ReactNode }) => (
  <p className="px-3 pb-1.5 pt-5 text-[11px] font-medium uppercase tracking-widest text-muted-foreground/70">
    {children}
  </p>
);

export function AppSidebar() {
  const { CONTROL_PLANE_ENABLED } = useSettings();
  const { isAdmin } = useCurrentUser();
  const { apps, selectedAppId, setSelectedAppId, refreshApps, isLoading } = useSelectedApp();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

  const handleAppCreated = async (newAppId: string) => {
    await refreshApps();
    setSelectedAppId(newAppId);
  };

  return (
    <>
      <aside className="sticky top-0 flex h-screen w-64 shrink-0 flex-col border-r bg-background">
        <div className="flex items-center gap-2.5 px-5 pb-2 pt-5">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Radio className="h-4 w-4" strokeWidth={2} />
          </div>
          <span className="text-[15px] font-semibold tracking-tight">Expo Open OTA</span>
        </div>

        <div className="px-3 pt-3">
          {/* Always rendered, even with a single app: the selector is what tells
              you which app every view below is scoped to. Creating apps only
              exists on the control plane and is an admin action, so the action
              is gated on both. */}
          <Combobox
            className="h-10 w-full rounded-lg"
            label="Select app"
            options={apps.map(a => ({ value: a.id, label: a.name || a.id }))}
            value={selectedAppId ?? ''}
            onChange={v => {
              if (v) setSelectedAppId(v);
            }}
            loading={isLoading}
            action={
              CONTROL_PLANE_ENABLED && isAdmin
                ? {
                    label: 'New application',
                    icon: <Plus className="mr-2 h-4 w-4" />,
                    onSelect: () => setIsCreateModalOpen(true),
                  }
                : undefined
            }
          />
        </div>

        <nav className="flex-1 overflow-y-auto px-3">
          {/* App-scoped pages are meaningless without a selected app (fresh
              control-plane install with no app yet) — hide the whole section
              until one is selected. */}
          {selectedAppId && (
            <>
              <SectionLabel>Application</SectionLabel>
              <div className="space-y-0.5">
                <NavLink to="/" icon={HardDriveDownload}>
                  Updates
                </NavLink>
                <NavLink to="/channels" icon={Box}>
                  Channels
                </NavLink>
                <NavLink to="/app-info" icon={Info}>
                  App info
                </NavLink>
                {CONTROL_PLANE_ENABLED && (
                  <NavLink to="/tokens" icon={KeyRound}>
                    API tokens
                  </NavLink>
                )}
              </div>

              <div className="mx-3 mt-5 border-t" />
            </>
          )}

          <SectionLabel>Server</SectionLabel>
          <div className="space-y-0.5">
            <NavLink to="/settings" icon={Settings}>
              Settings
            </NavLink>
            {CONTROL_PLANE_ENABLED && isAdmin && (
              <NavLink to="/users" icon={Users}>
                Users
              </NavLink>
            )}
            <NavLink to="/account" icon={CircleUser}>
              My account
            </NavLink>
          </div>
        </nav>

        <div className="border-t p-3">
          <Link
            to="/logout"
            className="flex items-center gap-2.5 rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-muted/60 hover:text-foreground">
            <LogOut className="h-4 w-4" strokeWidth={1.75} />
            <span>Log out</span>
          </Link>
        </div>
      </aside>

      {CONTROL_PLANE_ENABLED && isAdmin && (
        <CreateAppModal
          isOpen={isCreateModalOpen}
          onClose={() => setIsCreateModalOpen(false)}
          onAppCreated={handleAppCreated}
        />
      )}
    </>
  );
}
