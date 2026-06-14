import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api, ApiProblemError, BranchRecord } from '@/lib/api.ts';
import { ApiError } from '@/components/APIError';
import { DataTable } from '@/components/DataTable';
import { Box, GitBranch, Plus, Trash2 } from 'lucide-react';
import { useSearchParams } from 'react-router';
import { useSelectedApp } from '@/lib/SelectedAppContext';
import { useMemo, useState } from 'react';
import { Button } from '@/components/ui/button';
import { useToast } from '@/hooks/use-toast.ts';
import { ResourceCreateForm } from '@/components/ui/resource-create-form';
import { DeleteDialog } from '@/components/ui/delete-dialog';
import { useSettings } from '@/lib/SettingsContext';

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
        description: `Branch "${newBranchName.trim()}" initialized successfully.`,
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
        description: `Branch "${branchToDelete.branchName}" and all associated update items dropped successfully.`,
      });
      setBranchToDelete(null);
    } catch (error) {
      let errorTitle = 'Deletion failed';
      let errorMessage = 'Failed to destroy selected tracking branch.';
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
        header: 'Branch name',
        accessorKey: 'branchName',
        cell: ({ row }) => {
          return (
            <button
              className="flex flex-row gap-2 items-center cursor-pointer w-full"
              onClick={() => {
                setSearchParams({ branch: row.original.branchName });
              }}>
              <GitBranch className="w-4" />
              <span className="underline">{row.original.branchName}</span>
            </button>
          );
        },
      },
      {
        header: 'Release channel',
        size: 10,
        maxSize: 10,
        accessorKey: 'releaseChannel',
        cell: ({ row }) => {
          const releaseChannel = row.original.releaseChannel;
          if (!releaseChannel) return <span>N/A</span>;
          return (
            <div className="flex flex-row gap-2 items-center">
              <Box className="w-4" />
              <span>{row.original.releaseChannel}</span>
            </div>
          );
        },
      },
      ...(CONTROL_PLANE_ENABLED
        ? [
            {
              header: '',
              id: 'actions',
              cell: ({ row }) => {
                return (
                  <div className="text-right pr-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setBranchToDelete(row.original)}
                      className="h-8 w-8 p-0 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                      title="Purge Branch"
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
  }, [CONTROL_PLANE_ENABLED, setSearchParams]);

  return (
    <div className="w-full flex-1 space-y-6">
      {!!error && <ApiError error={error} />}
      
      {CONTROL_PLANE_ENABLED && (
        <ResourceCreateForm
          id="branch-name"
          label="New Release Branch Name"
          placeholder="e.g., main, feature-v3"
          inputValue={newBranchName}
          onInputChange={setNewBranchName}
          onSubmit={() => handleCreateBranch({ preventDefault: () => {} } as React.FormEvent)}
          isSubmitting={isCreating}
          buttonText="Provision Branch"
          icon={Plus}
        />
      )}

      <DataTable
        loading={isLoading}
        columns={tableColumns}
        data={data ?? []}
      />

      {CONTROL_PLANE_ENABLED && (
        <DeleteDialog
          isOpen={!!branchToDelete}
          onClose={() => setBranchToDelete(null)}
          onConfirm={handleExecuteDeletion}
          isDeleting={isDeleting}
          title="Delete Target Branch"
          resourceName={branchToDelete?.branchName}
          descriptionText="Warning: This operation executes a cascading delete. All nested compiled OTA update bundles assigned to this branch reference will be permanently destroyed on the server."
          confirmButtonText="Delete Branch"
          isDeletingButtonText="Deleting..."
        />
      )}
    </div>
  );
};