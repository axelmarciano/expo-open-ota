import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api, ApiProblemError, BranchRecord } from '@/lib/api.ts';
import { ApiError } from '@/components/APIError';
import { DataTable } from '@/components/DataTable';
import { GitBranch, Plus, Trash2 } from 'lucide-react';
import { useSearchParams } from 'react-router';
import { useSelectedApp } from '@/lib/SelectedAppContext';
import { useMemo, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useToast } from '@/hooks/use-toast.ts';
import { ResourceCreateForm } from '@/components/ui/resource-create-form';
import { DeleteDialog } from '@/components/ui/delete-dialog';
import { AdminOnlyNote } from '@/components/ui/admin-only-note';
import { useSettings } from '@/lib/SettingsContext';
import { useCurrentUser } from '@/lib/CurrentUserContext';

interface TableColumnConfig {
  header: string;
  accessorKey?: keyof BranchRecord;
  id?: string;
  size?: number;
  maxSize?: number;
  cell: (props: { row: { original: BranchRecord } }) => React.ReactNode;
}

export const BranchesTable = () => {
  const { CONTROL_PLANE_ENABLED } = useSettings();
  // Member accounts are read-only; every mutation is admin-only server-side.
  const { isAdmin } = useCurrentUser();
  const [, setSearchParams] = useSearchParams();
  const { selectedAppId } = useSelectedApp();
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const [newBranchName, setNewBranchName] = useState('');
  const [isCreating, setIsCreating] = useState(false);

  const [branchToDelete, setBranchToDelete] = useState<BranchRecord | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const { data, isLoading, error } = useQuery({
    queryKey: ['branches', selectedAppId],
    queryFn: () => api.getBranches(),
    enabled: !!selectedAppId,
  });

  const handleCreateBranch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newBranchName.trim()) return;

    setIsCreating(true);
    try {
      await api.createBranch(newBranchName.trim());
      setNewBranchName('');
      queryClient.invalidateQueries({ queryKey: ['branches', selectedAppId] });
      toast({
        title: 'Branch created',
        description: `"${newBranchName.trim()}" is ready to receive updates.`,
      });
    } catch (error) {
      let errorTitle = 'Error creating branch';
      let errorMessage = 'An unexpected error occurred.';
      if (error instanceof ApiProblemError) {
        errorTitle = error.title;
        errorMessage = error.detail;
      } else if (error instanceof Error) {
        errorMessage = error.message;
      }
      toast({
        title: errorTitle,
        description: errorMessage,
        variant: 'destructive',
      });
    } finally {
      setIsCreating(false);
    }
  };

  const handleExecuteDeletion = async () => {
    if (!branchToDelete) return;
    setIsDeleting(true);
    try {
      await api.deleteBranch(branchToDelete.branchName);
      queryClient.invalidateQueries({ queryKey: ['branches', selectedAppId] });
      toast({
        title: 'Branch deleted',
        description: `"${branchToDelete.branchName}" and its updates were removed.`,
      });
      setBranchToDelete(null);
    } catch (error) {
      let errorTitle = 'Deletion failed';
      let errorMessage = 'Could not delete the branch.';
      if (error instanceof ApiProblemError) {
        errorTitle = error.title;
        errorMessage = error.detail;
      } else if (error instanceof Error) {
        errorMessage = error.message;
      }
      toast({
        title: errorTitle,
        description: errorMessage,
        variant: 'destructive',
      });
    } finally {
      setIsDeleting(false);
    }
  };

  const tableColumns = useMemo<TableColumnConfig[]>(() => {
    return [
      {
        header: 'Branch',
        accessorKey: 'branchName',
        cell: ({ row }) => (
          <span className="flex items-center gap-2.5">
            <span className="flex h-7 w-7 items-center justify-center rounded-md bg-muted text-muted-foreground">
              <GitBranch className="h-3.5 w-3.5" />
            </span>
            <span className="font-medium">{row.original.branchName}</span>
          </span>
        ),
      },
      {
        header: 'Release channel',
        accessorKey: 'releaseChannel',
        cell: ({ row }) => {
          const releaseChannel = row.original.releaseChannel;
          if (!releaseChannel) {
            return <span className="text-muted-foreground/60">Not mapped</span>;
          }
          return <Badge variant="outline">{releaseChannel}</Badge>;
        },
      },
      ...(CONTROL_PLANE_ENABLED && isAdmin
        ? [
            {
              header: '',
              id: 'actions',
              cell: ({ row }) => {
                return (
                  <div className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={e => {
                        e.stopPropagation();
                        setBranchToDelete(row.original);
                      }}
                      className="h-8 w-8 p-0 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                      title="Delete branch"
                    >
                      <Trash2 />
                    </Button>
                  </div>
                );
              },
            } satisfies TableColumnConfig,
          ]
        : []),
    ];
  }, [CONTROL_PLANE_ENABLED, isAdmin]);

  return (
    <div className="w-full flex-1 space-y-4">
      {!!error && <ApiError error={error} />}

      {CONTROL_PLANE_ENABLED && !isAdmin && (
        <AdminOnlyNote>
          You are signed in with a member account, which is read-only. Ask an admin to create or
          delete branches.
        </AdminOnlyNote>
      )}
      {CONTROL_PLANE_ENABLED && isAdmin && (
        <div className="flex justify-end">
          <ResourceCreateForm
            id="branch-name"
            label="New branch name"
            placeholder="New branch name, e.g. main"
            inputValue={newBranchName}
            onInputChange={setNewBranchName}
            onSubmit={() => handleCreateBranch({ preventDefault: () => {} } as React.FormEvent)}
            isSubmitting={isCreating}
            buttonText="Create branch"
            icon={Plus}
          />
        </div>
      )}

      <DataTable
        loading={isLoading}
        columns={tableColumns}
        data={data ?? []}
        emptyMessage="No branches yet. Publish an update or create a branch to get started."
        onRowClick={row => setSearchParams({ branch: row.branchName })}
      />

      {CONTROL_PLANE_ENABLED && isAdmin && (
        <DeleteDialog
          isOpen={!!branchToDelete}
          onClose={() => setBranchToDelete(null)}
          onConfirm={handleExecuteDeletion}
          isDeleting={isDeleting}
          title="Delete branch"
          resourceName={branchToDelete?.branchName}
          descriptionText="Every update published to this branch will be permanently deleted. Apps pointing at a channel mapped to this branch will stop receiving updates."
          confirmButtonText="Delete branch"
          isDeletingButtonText="Deleting…"
        />
      )}
    </div>
  );
};
