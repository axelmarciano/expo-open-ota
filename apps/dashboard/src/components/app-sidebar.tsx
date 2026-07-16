import { useState } from 'react';
import { Link, useLocation } from 'react-router';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarHeader,
} from '@/components/ui/sidebar';
import { Box, HardDriveDownload, PowerOff, Settings, Plus, Info } from 'lucide-react';
import clsx from 'clsx';
import { Combobox } from '@/components/Combobox';
import { useSelectedApp } from '@/lib/SelectedAppContext';
import { CreateAppModal } from '@/components/app-creation-modal';
import { useSettings } from '@/lib/SettingsContext';

const items = [
  {
    title: 'Updates',
    url: '/',
    icon: HardDriveDownload,
  },
  {
    title: 'Channels',
    url: '/channels',
    icon: Box,
  },
  {
    title: 'App Info',
    url: '/app-info',
    icon: Info,
  },
  {
    title: 'Settings',
    url: '/settings',
    icon: Settings,
  },
  {
    title: 'Logout',
    url: '/logout',
    icon: PowerOff,
  },
];

export function AppSidebar() {
  const { CONTROL_PLANE_ENABLED } = useSettings();
  const location = useLocation();
  const currentPath = location.pathname;
  const { apps, selectedAppId, setSelectedAppId, refreshApps, isLoading } = useSelectedApp();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

  const handleAppCreated = async (newAppId: string) => {
    await refreshApps();
    setSelectedAppId(newAppId);
  };

  return (
    <>
      <Sidebar className="w-64 bg-white border-r border-gray-200">
        <SidebarHeader className="p-4 border-b">
          <h1 className="text-lg font-semibold">Expo Open OTA</h1>
        </SidebarHeader>
        <SidebarContent className="p-2">
          {/* Always rendered, even with a single app: the selector is what tells
              you which app every view below is scoped to. Creating apps only
              exists on the control plane, so the action is gated on it. */}
          <SidebarGroup>
            <SidebarGroupLabel>App</SidebarGroupLabel>
            <SidebarGroupContent className="px-2 pb-2">
              <Combobox
                label="Select app"
                options={apps.map(a => ({ value: a.id, label: a.name || a.id }))}
                value={selectedAppId ?? ''}
                onChange={v => {
                  if (v) setSelectedAppId(v);
                }}
                loading={isLoading}
                action={
                  CONTROL_PLANE_ENABLED
                    ? {
                        label: 'New Application',
                        icon: <Plus className="mr-2 h-4 w-4" />,
                        onSelect: () => setIsCreateModalOpen(true),
                      }
                    : undefined
                }
              />
            </SidebarGroupContent>
          </SidebarGroup>

          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                {items.map(item => {
                  const isActive = currentPath === item.url;
                  return (
                    <SidebarMenuItem key={item.title}>
                      <SidebarMenuButton asChild disabled={isActive}>
                        <Link
                          to={item.url}
                          onClick={e => {
                            if (isActive) {
                              e.preventDefault();
                            }
                          }}
                          className={clsx(
                            'flex items-center gap-2 px-4 py-2 rounded-lg transition',
                            isActive ? 'bg-gray-200 text-black' : 'text-gray-500 hover:bg-gray-100'
                          )}>
                          <item.icon className="w-5 h-5" />
                          <span>{item.title}</span>
                        </Link>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  );
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>
        <SidebarFooter />
      </Sidebar>

      {CONTROL_PLANE_ENABLED && (
        <CreateAppModal 
          isOpen={isCreateModalOpen}
          onClose={() => setIsCreateModalOpen(false)}
          onAppCreated={handleAppCreated}
        />
      )}
    </>
  );
}