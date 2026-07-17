import { useState } from 'react';
import { Eye, Plus, Trash2 } from 'lucide-react';
import { ApiKeyRecord } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { ResourceCreateForm } from '@/components/ui/resource-create-form';
import { DataTable } from '@/components/DataTable';
import { TimestampCell } from '@/components/ui/timestamp-cell';

type ApiTokensManagerProps = {
  isLoading: boolean;
  keysList: ApiKeyRecord[];
  onCreateKey: (keyName: string) => Promise<void>;
  isCreatingKey: boolean;
  onSelectKeyToRevoke: (key: ApiKeyRecord) => void;
  generatedToken: string | null;
  onClearGeneratedToken: () => void;
};

export const ApiTokensManager = ({
  isLoading,
  keysList,
  onCreateKey,
  isCreatingKey,
  onSelectKeyToRevoke,
  generatedToken,
  onClearGeneratedToken,
}: ApiTokensManagerProps) => {
  const [newKeyName, setNewKeyName] = useState('');

  const handleFormSubmit = async (keyName: string) => {
    await onCreateKey(keyName);
    setNewKeyName('');
  };

  return (
    <div className="lg:col-span-2 space-y-4">
      <ResourceCreateForm
        id="token-name"
        label="Token Description Name"
        placeholder="e.g., CI/CD Pipeline Deployment Integration Key"
        inputValue={newKeyName}
        onInputChange={setNewKeyName}
        onSubmit={(e) => {
          e.preventDefault();
          handleFormSubmit(newKeyName);
        }}
        isSubmitting={isCreatingKey}
        buttonText="Create Token"
        icon={Plus}
      />

      {generatedToken && (
        <Card className="border-primary/20 bg-muted/40 overflow-hidden">
          <CardContent className="p-4 space-y-2">
            <div className="flex items-center gap-1.5 text-xs font-medium text-foreground">
              <Eye className="w-3.5 h-3.5 text-primary" />
              <span>API Secret Key Issued (Copy this now — it will not be shown again):</span>
            </div>
            <div className="flex items-center gap-2">
              <code className="text-xs font-mono select-all bg-background border border-border text-foreground p-2 rounded flex-1 break-all">
                {generatedToken}
              </code>
              <Button 
                variant="ghost" 
                size="sm" 
                onClick={onClearGeneratedToken}
                className="text-xs text-muted-foreground hover:text-foreground"
              >
                Dismiss
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      <DataTable
        loading={isLoading}
        columns={[
          {
            header: 'Token Name Identifier',
            accessorKey: 'name',
            cell: ({ row }) => (
              <span className="text-sm font-medium">
                {row.original.name}
              </span>
            ),
          },
          {
            header: 'Prefix Hint',
            accessorKey: 'hint',
            cell: ({ row }) => (
              <span className="font-mono text-muted-foreground">
                {row.original.hint}
              </span>
            ),
          },
          {
            header: 'Provisioned Date',
            accessorKey: 'createdAt',
            cell: ({ row }) => (
              <TimestampCell dateString={row.original.createdAt} />
            ),
          },
          {
            header: 'Last Used At',
            accessorKey: 'lastUsedAt',
            cell: ({ row }) => {
              const lastUsed = row.original.lastUsedAt;
              if (!lastUsed) {
                return <span className="text-muted-foreground/60 italic">Never used</span>;
              }
              return <TimestampCell dateString={lastUsed} />;
            },
          },
          {
            header: '',
            id: 'actions',
            cell: ({ row }) => (
              <div className="text-right pr-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onSelectKeyToRevoke(row.original)}
                  className="h-8 w-8 p-0 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                  title="Revoke Token Access"
                >
                  <Trash2 />
                </Button>
              </div>
            ),
          },
        ]}
        data={keysList}
      />
    </div>
  );
};