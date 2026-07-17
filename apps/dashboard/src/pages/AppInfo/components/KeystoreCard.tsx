import { Key } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { AppDetails } from '@/lib/api';

type KeystoreCardProps = {
  isLoading: boolean;
  appData: AppDetails | undefined;
};

export const KeystoreCard = ({ isLoading, appData }: KeystoreCardProps) => {
  return (
    <Card className="lg:col-span-1">
      <CardContent className="p-4 space-y-4">
        <div className="flex items-center gap-2 pb-2 border-b border-border">
          <Key className="w-4 h-4 text-muted-foreground" />
          <h3 className="font-medium text-sm">Keystore Configuration</h3>
        </div>

        {isLoading ? (
          <div className="space-y-2 py-2 animate-pulse">
            <div className="h-3 bg-muted rounded w-1/3" />
            <div className="h-8 bg-muted rounded" />
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-1">
              <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider block">
                Storage Architecture Mode
              </span>
              <Badge variant="secondary" className="capitalize">
                {appData?.keys?.mode?.replace('-', ' ') || 'Database Managed'}
              </Badge>
            </div>

            {appData?.keys?.mode === 'local' && (
              <div className="space-y-3 pt-2 border-t border-border">
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Public Key Reference Path</Label>
                  <p className="text-xs font-mono text-foreground break-all bg-muted/50 p-2 rounded border border-border">
                    {appData.keys.publicPath || 'None configured'}
                  </p>
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Private Key Reference Path</Label>
                  <p className="text-xs font-mono text-foreground break-all bg-muted/50 p-2 rounded border border-border">
                    {appData.keys.privatePath || 'None configured'}
                  </p>
                </div>
              </div>
            )}

            {appData?.keys?.mode === 'aws-secrets-manager' && (
              <div className="space-y-3 pt-2 border-t border-border">
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">AWS Secret ID (Public)</Label>
                  <p className="text-xs font-mono text-foreground break-all bg-muted/50 p-2 rounded border border-border">
                    {appData.keys.publicSecretId || 'None configured'}
                  </p>
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">AWS Secret ID (Private)</Label>
                  <p className="text-xs font-mono text-foreground break-all bg-muted/50 p-2 rounded border border-border">
                    {appData.keys.privateSecretId || 'None configured'}
                  </p>
                </div>
              </div>
            )}

            {appData?.keys?.mode === 'database' && (
              <p className="text-xs text-muted-foreground leading-relaxed">
                Cryptographic verification signing keys are handled inside native database stores. No external reference configurations are required.
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
};