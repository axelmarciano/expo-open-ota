import { useState, useEffect } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api, ApiKeyRecord, ApiProblemError } from '@/lib/api';
import { useSelectedApp } from '@/lib/SelectedAppContext';
import { useToast } from '@/hooks/use-toast';
import { ShieldAlert, Download, Edit2, Check, X, Trash2, AlertOctagon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
import { Card, CardContent } from '@/components/ui/card';
import { DeleteDialog } from '@/components/ui/delete-dialog';
import { KeystoreCard } from './components/KeystoreCard';
import { ApiTokensManager } from './components/ApiTokensManager';
import { useSettings } from '@/lib/SettingsContext';

export const AppInfo = () => {
  const { CONTROL_PLANE_ENABLED } = useSettings();
  const { selectedAppId, refreshApps } = useSelectedApp();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  
  const [generatedToken, setGeneratedToken] = useState<string | null>(null);
  const [isCreatingKey, setIsCreatingKey] = useState(false);
  const [keyToRevoke, setKeyToRevoke] = useState<ApiKeyRecord | null>(null);
  const [isRevokingKey, setIsRevokingKey] = useState(false);

  const [isEditingName, setIsEditingName] = useState(false);
  const [editNameValue, setEditNameValue] = useState('');
  const [isUpdatingName, setIsUpdatingName] = useState(false);
  
  const [showDeleteAppDialog, setShowDeleteAppDialog] = useState(false);
  const [isDeletingApp, setIsDeletingApp] = useState(false);

  const appDetailsQuery = useQuery({
    queryKey: ['appDetails', selectedAppId],
    queryFn: () => api.getApp(selectedAppId!),
    enabled: !!selectedAppId,
  });

  const apiKeysQuery = useQuery({
    queryKey: ['apiKeys', selectedAppId],
    queryFn: () => api.getApiKeys(),
    enabled: !!selectedAppId && CONTROL_PLANE_ENABLED,
  });

  const appData = appDetailsQuery.data;
  const keysList = apiKeysQuery.data || [];
  const isDownloadAvailable = appData?.keys?.mode === 'database';

  useEffect(() => {
    if (appData?.name) {
      setEditNameValue(appData.name);
    }
  }, [appData?.name]);

  if (!selectedAppId) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[400px] border border-dashed rounded-xl p-8 text-center bg-background">
        <ShieldAlert className="w-8 h-8 text-muted-foreground mb-2" />
        <p className="text-sm font-medium text-foreground">No application selected</p>
        <p className="text-xs text-muted-foreground mt-0.5">Please choose or register an application via the dashboard sidebar.</p>
      </div>
    );
  }

  const handleUpdateAppName = async () => {
    if (!editNameValue.trim() || editNameValue.trim() === appData?.name) {
      setIsEditingName(false);
      return;
    }
    setIsUpdatingName(true);
    try {
      await api.updateApp({ name: editNameValue.trim() });
      await queryClient.invalidateQueries({ queryKey: ['appDetails', selectedAppId] });
      toast({ title: 'Success', description: 'Application name modified.' });
      setIsEditingName(false);
      await refreshApps();
    } catch (error) {
      let errorTitle = 'Error updating application name';
      let errorMessage = 'An unexpected server error occurred.';
      if (error instanceof ApiProblemError) {
        errorTitle = error.title;
        errorMessage = error.detail;
      }
      else if (error instanceof Error) errorMessage = error.message;
      toast({
        title: errorTitle,
        description: errorMessage,
        variant: 'destructive',
      });
    } finally {
      setIsUpdatingName(false);
    }
  };

const handleExecuteAppDeletion = async () => {
    setIsDeletingApp(true);
    try {
      await api.deleteApp();
      toast({ 
        title: 'Application Deleted', 
        description: 'The configuration workspace was successfully removed.' 
      });
      setShowDeleteAppDialog(false);
      setIsDeletingApp(false);
      await refreshApps();
    } catch (error) {
      let errorTitle = 'Deletion failed';
      let errorMessage = 'Could not delete application.';
      if (error instanceof ApiProblemError) {
        errorMessage = error.detail;
        errorTitle = error.title;
      } else if (error instanceof Error) errorMessage = error.message;
      toast({
        title: errorTitle,
        description: errorMessage,
        variant: 'destructive',
      });
      setIsDeletingApp(false);
    }
  };

  const handleCreateApiKey = async (keyName: string) => {
    setIsCreatingKey(true);
    try {
      const response = await api.createApiKey(keyName);
      setGeneratedToken(response.apiKey);
      queryClient.invalidateQueries({ queryKey: ['apiKeys', selectedAppId] });
      toast({ title: 'Success', description: 'API Key issued successfully.' });
    } catch (error) {
      let errorTitle = 'Error generating key';
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
      setIsCreatingKey(false);
    }
  };

  const handleExecuteRevocation = async () => {
    if (!keyToRevoke) return;
    setIsRevokingKey(true);
    try {
      await api.revokeApiKey(keyToRevoke.id);
      queryClient.invalidateQueries({ queryKey: ['apiKeys', selectedAppId] });
      toast({ title: 'Token Revoked', description: 'Key reference removed from environment security layer.' });
      setKeyToRevoke(null);
    } catch (error) {
      let errorTitle = 'Revocation failed';
      let errorMessage = 'Could not reach credential server configuration.';
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
      setIsRevokingKey(false);
    }
  };

  const handleDownloadCertificate = async () => {
    try {
      const certificateText = await api.downloadAppCertificate(selectedAppId);
      const blob = new Blob([certificateText], { type: 'text/plain' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `app-${selectedAppId}-certificate.txt`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
      toast({ title: 'Success', description: 'App public key certificate downloaded.' });
    } catch (error) {
      let errorTitle = 'Download failed';
      let errorMessage = 'Failed to download certificate.';
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
    }
  };

  return (
    <div className="w-full h-screen flex-1 p-5 space-y-6">
      <div className="space-y-2">
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
          <div className="space-y-2 flex-1">
            {isEditingName ? (
              <div className="flex items-center gap-2 max-w-md">
                <Input
                  value={editNameValue}
                  onChange={(e) => setEditNameValue(e.target.value)}
                  disabled={isUpdatingName}
                  className="font-medium"
                  autoFocus
                  onKeyDown={(e) => e.key === 'Enter' && handleUpdateAppName()}
                />
                <Button size="icon" variant="outline" className="h-9 w-9 text-green-600 hover:text-green-700 shrink-0" onClick={handleUpdateAppName} disabled={isUpdatingName}>
                  <Check className="w-4 h-4" />
                </Button>
                <Button size="icon" variant="ghost" className="h-9 w-9 text-muted-foreground shrink-0" onClick={() => { setIsEditingName(false); setEditNameValue(appData?.name || ''); }} disabled={isUpdatingName}>
                  <X className="w-4 h-4" />
                </Button>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <h1 className="text-2xl font-medium tracking-tight">
                  {appData?.name || selectedAppId}
                </h1>
                {CONTROL_PLANE_ENABLED && (
                  <Button 
                    variant="ghost" 
                    size="icon" 
                    className="h-7 w-7 text-muted-foreground hover:text-foreground transition-colors duration-150 shrink-0" 
                    onClick={() => setIsEditingName(true)}
                  >
                    <Edit2 className="w-3.5 h-3.5" />
                  </Button>
                )}
              </div>
            )}
            <p className="text-sm text-muted-foreground">
              Application Environment Identification ID:{' '}
              <code className="bg-muted text-foreground px-1.5 py-0.5 rounded text-xs font-mono">
                {selectedAppId}
              </code>
            </p>
          </div>

          {isDownloadAvailable && (
            <Button 
              variant="outline" 
              size="sm"
              onClick={handleDownloadCertificate}
              className="flex items-center gap-1.5 self-start md:self-auto"
            >
              <Download className="w-4 h-4" />
              Download Certificate
            </Button>
          )}
        </div>
        <Separator />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        <div className="lg:col-span-1 space-y-6">
          <KeystoreCard 
            isLoading={appDetailsQuery.isLoading} 
            appData={appData} 
          />
          {CONTROL_PLANE_ENABLED && (
            <Card className="border-destructive/20 bg-destructive/[0.01]">
              <CardContent className="p-4 space-y-4">
                <div className="flex items-center gap-2 pb-2 border-b border-destructive/10 text-destructive">
                  <AlertOctagon className="w-4 h-4" />
                  <h3 className="font-medium text-sm">Danger Zone</h3>
                </div>
                <div className="space-y-3">
                  <p className="text-xs text-muted-foreground leading-relaxed">
                    Permanently wipe out this complete application tracking workspace along with all branches, channels, API validation profiles, and historical OTA build parameters.
                  </p>
                  <Button
                    variant="destructive"
                    size="sm"
                    className="w-full text-xs"
                    onClick={() => setShowDeleteAppDialog(true)}
                  >
                    <Trash2 className="w-3.5 h-3.5 mr-1.5" /> Delete Application
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
                
        {CONTROL_PLANE_ENABLED ? (
          <ApiTokensManager
            isLoading={apiKeysQuery.isLoading}
            keysList={keysList}
            onCreateKey={handleCreateApiKey}
            isCreatingKey={isCreatingKey}
            onSelectKeyToRevoke={setKeyToRevoke}
            generatedToken={generatedToken}
            onClearGeneratedToken={() => setGeneratedToken(null)}
          />
        ) : (
          <div className="lg:col-span-2 text-sm text-muted-foreground italic border border-dashed rounded-xl p-6 bg-muted/20 text-center">
            API tokens are handled statically via configuration files on this self-hosted platform instance.
          </div>
        )}
      </div>

      {CONTROL_PLANE_ENABLED && (
        <DeleteDialog
          isOpen={!!keyToRevoke}
          onClose={() => setKeyToRevoke(null)}
          onConfirm={handleExecuteRevocation}
          isDeleting={isRevokingKey}
          title="Revoke Access Token"
          resourceName={keyToRevoke?.name}
          descriptionText="Any active external systems or deployment integrations using this signature configuration string will immediately fail authentication requests. This action is permanent."
          confirmButtonText="Revoke Token"
          isDeletingButtonText="Revoking..."
        />
      )}

      {CONTROL_PLANE_ENABLED && (
        <DeleteDialog
          isOpen={showDeleteAppDialog}
          onClose={() => setShowDeleteAppDialog(false)}
          onConfirm={handleExecuteAppDeletion}
          isDeleting={isDeletingApp}
          title="Delete Entire Application Environment"
          resourceName={appData?.name || selectedAppId}
          descriptionText="This triggers an absolute workspace data teardown. All deployment pipelines will permanently drop connectivity, and remote devices targeting this ecosystem registry scope string will immediately lose access to OTA updates."
          confirmButtonText="Delete Application Space"
          isDeletingButtonText="Deleting Application Space..."
        />
      )}
    </div>
  );
};