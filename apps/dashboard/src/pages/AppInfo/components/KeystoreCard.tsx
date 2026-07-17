import { Key } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { AppDetails } from '@/lib/api';

type KeystoreCardProps = {
  isLoading: boolean;
  appData: AppDetails | undefined;
};

const KeyPathRow = ({ label, value }: { label: string; value?: string }) => (
  <div className="space-y-1">
    <p className="text-xs text-muted-foreground">{label}</p>
    <p className="break-all rounded-lg border bg-muted/40 p-2 font-mono text-xs text-foreground">
      {value || 'Not configured'}
    </p>
  </div>
);

export const KeystoreCard = ({ isLoading, appData }: KeystoreCardProps) => {
  return (
    <Card>
      <CardContent className="space-y-4 p-5">
        <div className="flex items-center gap-2">
          <Key className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-medium">Signing keys</h3>
        </div>

        {isLoading ? (
          <div className="animate-pulse space-y-2 py-2">
            <div className="h-3 w-1/3 rounded bg-muted" />
            <div className="h-8 rounded bg-muted" />
          </div>
        ) : (
          <div className="space-y-4">
            <div className="flex items-center gap-2 text-sm">
              <span className="text-muted-foreground">Stored in</span>
              <Badge variant="secondary" className="capitalize">
                {appData?.keys?.mode?.replace(/-/g, ' ') || 'database'}
              </Badge>
            </div>

            {appData?.keys?.mode === 'local' && (
              <div className="space-y-3 border-t pt-4">
                <KeyPathRow label="Public key path" value={appData.keys.publicPath} />
                <KeyPathRow label="Private key path" value={appData.keys.privatePath} />
              </div>
            )}

            {appData?.keys?.mode === 'aws-secrets-manager' && (
              <div className="space-y-3 border-t pt-4">
                <KeyPathRow label="Public key secret ID" value={appData.keys.publicSecretId} />
                <KeyPathRow label="Private key secret ID" value={appData.keys.privateSecretId} />
              </div>
            )}

            {appData?.keys?.mode === 'database' && (
              <p className="text-xs leading-relaxed text-muted-foreground">
                The key pair used to sign updates is stored encrypted in the database — nothing to
                configure on your side.
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
};
