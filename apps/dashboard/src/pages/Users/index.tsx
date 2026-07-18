import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { KeyRound, Plus, Shield, ShieldOff, Trash2 } from 'lucide-react';
import { api, UserRecord, describeApiError } from '@/lib/api';
import { useSettings } from '@/lib/SettingsContext';
import { useCurrentUser } from '@/lib/CurrentUserContext';
import { useToast } from '@/hooks/use-toast';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/PageHeader';
import { DataTable } from '@/components/DataTable';
import { TimestampCell } from '@/components/ui/timestamp-cell';
import { DeleteDialog } from '@/components/ui/delete-dialog';
import { CreateUserModal } from '@/components/user-creation-modal';

export const Users = () => {
  const { CONTROL_PLANE_ENABLED, SSO_ENABLED } = useSettings();
  const { user: currentUser, isAdmin } = useCurrentUser();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [userToDelete, setUserToDelete] = useState<UserRecord | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [togglingUserId, setTogglingUserId] = useState<string | null>(null);

  const usersQuery = useQuery({
    queryKey: ['users'],
    queryFn: () => api.getUsers(),
    enabled: CONTROL_PLANE_ENABLED && isAdmin,
  });

  const notifyError = (error: unknown, fallbackTitle: string) =>
    toast({ ...describeApiError(error, fallbackTitle), variant: 'destructive' });

  const handleToggleAdmin = async (user: UserRecord) => {
    setTogglingUserId(user.id);
    try {
      await api.updateUserAdmin(user.id, !user.isAdmin);
      queryClient.invalidateQueries({ queryKey: ['users'] });
      toast({
        title: user.isAdmin ? 'Admin removed' : 'Admin granted',
        description: `"${user.email}" is ${user.isAdmin ? 'no longer' : 'now'} an admin.`,
      });
    } catch (error) {
      notifyError(error, 'Error updating user');
    } finally {
      setTogglingUserId(null);
    }
  };

  const handleExecuteDeletion = async () => {
    if (!userToDelete) return;
    setIsDeleting(true);
    try {
      await api.deleteUser(userToDelete.id);
      queryClient.invalidateQueries({ queryKey: ['users'] });
      toast({ title: 'User deleted', description: `"${userToDelete.email}" can no longer sign in.` });
      setUserToDelete(null);
    } catch (error) {
      notifyError(error, 'Deletion failed');
    } finally {
      setIsDeleting(false);
    }
  };

  if (!CONTROL_PLANE_ENABLED) {
    return (
      <div className="w-full">
        <PageHeader
          title="Users"
          description="User accounts for this dashboard."
        />
        <div className="rounded-xl border border-dashed bg-muted/30 p-8 text-center text-sm text-muted-foreground">
          On a stateless deployment there is a single account, configured through the ADMIN_EMAIL
          and ADMIN_PASSWORD environment variables.
        </div>
      </div>
    );
  }

  if (!isAdmin) {
    return (
      <div className="w-full">
        <PageHeader title="Users" description="User accounts for this dashboard." />
        <div className="rounded-xl border border-dashed bg-muted/30 p-8 text-center text-sm text-muted-foreground">
          Only admins can manage users. Ask an admin if you need access.
        </div>
      </div>
    );
  }

  return (
    <div className="w-full">
      <PageHeader
        title="Users"
        description="Accounts that can sign in to this dashboard. Admins can manage users, create apps and remap release channels; you cannot change your own admin status, and there must always be at least one admin."
      />

      <div className="space-y-4">
        {/* While SSO is active, accounts arrive through provisioning on first
            sign-in; the server refuses manual creation anyway. */}
        {SSO_ENABLED ? (
          <div className="flex items-center gap-2 rounded-lg border bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
            <KeyRound className="h-4 w-4 shrink-0" />
            Single sign-on is active: accounts are created automatically the first time someone
            signs in with SSO. Promote or remove them from this page.
          </div>
        ) : (
          <div className="flex justify-end">
            <Button onClick={() => setIsCreateModalOpen(true)}>
              <Plus className="h-4 w-4" />
              Create user
            </Button>
          </div>
        )}

        <DataTable
          loading={usersQuery.isLoading}
          columns={[
            {
              header: 'Email',
              accessorKey: 'email',
              cell: ({ row }) => (
                <span className="font-medium">
                  {row.original.email}
                  {row.original.id === currentUser?.id && (
                    <span className="ml-2 text-xs text-muted-foreground">(you)</span>
                  )}
                </span>
              ),
            },
            {
              header: 'Role',
              accessorKey: 'isAdmin',
              cell: ({ row }) =>
                row.original.isAdmin ? (
                  <Badge>Admin</Badge>
                ) : (
                  <Badge variant="secondary">Member</Badge>
                ),
            },
            {
              header: 'Created',
              accessorKey: 'createdAt',
              cell: ({ row }) => <TimestampCell dateString={row.original.createdAt ?? null} />,
            },
            {
              header: 'Last connected',
              accessorKey: 'lastConnectedAt',
              cell: ({ row }) => {
                const lastConnected = row.original.lastConnectedAt;
                if (!lastConnected) {
                  return <span className="text-muted-foreground/60">Never</span>;
                }
                return <TimestampCell dateString={lastConnected} />;
              },
            },
            {
              header: '',
              id: 'actions',
              cell: ({ row }) => {
                // The server refuses self-service admin changes and
                // self-deletion — do not offer them.
                if (row.original.id === currentUser?.id) return null;
                return (
                  <div className="flex items-center justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleToggleAdmin(row.original)}
                      disabled={togglingUserId === row.original.id}
                      className="h-8 w-8 p-0 text-muted-foreground hover:text-foreground"
                      title={row.original.isAdmin ? 'Remove admin' : 'Make admin'}>
                      {row.original.isAdmin ? <ShieldOff /> : <Shield />}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setUserToDelete(row.original)}
                      className="h-8 w-8 p-0 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                      title="Delete user">
                      <Trash2 />
                    </Button>
                  </div>
                );
              },
            },
          ]}
          data={usersQuery.data ?? []}
          emptyMessage="No users yet."
        />
      </div>

      <CreateUserModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onUserCreated={() => queryClient.invalidateQueries({ queryKey: ['users'] })}
      />

      <DeleteDialog
        isOpen={!!userToDelete}
        onClose={() => setUserToDelete(null)}
        onConfirm={handleExecuteDeletion}
        isDeleting={isDeleting}
        title="Delete user"
        resourceName={userToDelete?.email}
        descriptionText="This account will no longer be able to sign in to the dashboard. This cannot be undone."
        confirmButtonText="Delete user"
        isDeletingButtonText="Deleting…"
      />
    </div>
  );
};
