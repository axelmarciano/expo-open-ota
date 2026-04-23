import { Link, useLocation } from 'react-router';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarHeader,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar';
import { Box, HardDriveDownload, PowerOff, Settings } from 'lucide-react';
import clsx from 'clsx';
import { Combobox } from '@/components/Combobox';
import { useSelectedApp } from '@/lib/SelectedAppContext';

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
  const location = useLocation();
  const currentPath = location.pathname;
  const { apps, selectedAppId, setSelectedAppId, isLoading } = useSelectedApp();

  // Only show the selector when there is something to choose from. Single-app
  // deployments (the majority) get the plain navigation with no extra UI.
  const showSelector = apps.length > 1;

  return (
    <Sidebar className="w-64 bg-white border-r border-gray-200">
      <SidebarHeader className="p-4 border-b">
        <h1 className="text-lg font-semibold">Expo Open OTA</h1>
      </SidebarHeader>
      <SidebarContent className="p-2">
        {showSelector && (
          <SidebarGroup>
            <SidebarGroupLabel>App</SidebarGroupLabel>
            <SidebarGroupContent className="px-2 pb-2">
              <Combobox
                label="Select app"
                options={apps.map(a => ({ value: a.id, label: a.name || a.id }))}
                value={selectedAppId ?? ''}
                onChange={v => {
                  // Empty string comes from Combobox's "toggle off" path —
                  // ignore it so the user can't end up with no selection.
                  if (v) setSelectedAppId(v);
                }}
                loading={isLoading}
              />
            </SidebarGroupContent>
          </SidebarGroup>
        )}
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
  );
}
