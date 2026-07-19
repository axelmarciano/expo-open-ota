// Copyright (c) 2026 Mercure Technologies. All rights reserved.
// This file is governed by the Expo Open OTA Enterprise Edition license
// (see ee/LICENSE at the repository root); it is NOT covered by the MIT
// license of this repository.

import { useEffect, useRef, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, ChevronRight, Copy, KeyRound, ShieldAlert, TriangleAlert, X } from 'lucide-react';
import { api, describeApiError, SsoSettings } from '@/lib/api';
import { useToast } from '@/hooks/use-toast';
import { cn } from '@/lib/utils';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { DeleteDialog } from '@/components/ui/delete-dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { Switch } from '@/components/ui/switch';

// The enabled toggle is deliberately not part of the form: it acts on the
// stored configuration immediately (its own PUT), while the form fields wait
// for "Save and test connection".
type SsoFormState = {
  issuer: string;
  clientId: string;
  clientSecret: string;
  providerName: string;
  scopes: string;
  groupsClaim: string;
  allowedEmailDomains: string[];
  allowedGroups: string[];
  trustUnverifiedEmail: boolean;
  manualUserValidation: boolean;
};

const DEFAULT_SCOPES = 'openid profile email';
const DEFAULT_GROUPS_CLAIM = 'groups';

const emptyForm: SsoFormState = {
  issuer: '',
  clientId: '',
  clientSecret: '',
  providerName: '',
  scopes: DEFAULT_SCOPES,
  groupsClaim: DEFAULT_GROUPS_CLAIM,
  allowedEmailDomains: [],
  allowedGroups: [],
  trustUnverifiedEmail: false,
  manualUserValidation: false,
};

const formFromSettings = (settings: SsoSettings): SsoFormState => ({
  issuer: settings.issuer,
  clientId: settings.clientId,
  // The secret never comes back from the server: an empty field means
  // "keep the stored one".
  clientSecret: '',
  providerName: settings.providerName,
  scopes: settings.scopes,
  groupsClaim: settings.groupsClaim,
  allowedEmailDomains: settings.allowedEmailDomains,
  allowedGroups: settings.allowedGroups,
  trustUnverifiedEmail: settings.trustUnverifiedEmail,
  manualUserValidation: settings.manualUserValidation,
});

// Signature of the stored values the form is populated from. The live enabled
// toggle rewrites the query cache; repopulating on it would wipe unsaved
// edits, so the form only resets when one of these actually changes.
const settingsFormSignature = (settings: SsoSettings | null): string | null =>
  settings
    ? JSON.stringify([
        settings.issuer,
        settings.clientId,
        settings.hasClientSecret,
        settings.providerName,
        settings.scopes,
        settings.groupsClaim,
        settings.allowedEmailDomains,
        settings.allowedGroups,
        settings.trustUnverifiedEmail,
        settings.manualUserValidation,
      ])
    : null;

// A small chips input: type a value, press Enter (or comma, or leave the
// field) to add it. Backspace on an empty field removes the last chip.
const TagListInput = ({
  id,
  values,
  onChange,
  placeholder,
}: {
  id: string;
  values: string[];
  onChange: (values: string[]) => void;
  placeholder?: string;
}) => {
  const [draft, setDraft] = useState('');

  const commitDraft = () => {
    const additions = draft
      .split(',')
      .map(part => part.trim())
      .filter(Boolean);
    if (!additions.length) {
      setDraft('');
      return;
    }
    const next = [...values];
    for (const addition of additions) {
      if (!next.includes(addition)) {
        next.push(addition);
      }
    }
    onChange(next);
    setDraft('');
  };

  return (
    <div className="space-y-2">
      {values.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {values.map(value => (
            <Badge key={value} variant="secondary" className="gap-1 pr-1 font-normal">
              <span className="max-w-[260px] truncate">{value}</span>
              <button
                type="button"
                aria-label={`Remove ${value}`}
                onClick={() => onChange(values.filter(entry => entry !== value))}
                className="rounded-full p-0.5 text-muted-foreground transition-colors hover:bg-muted-foreground/15 hover:text-foreground">
                <X className="h-3 w-3" />
              </button>
            </Badge>
          ))}
        </div>
      )}
      <Input
        id={id}
        value={draft}
        placeholder={placeholder}
        onChange={event => setDraft(event.target.value)}
        onKeyDown={event => {
          if (event.key === 'Enter' || event.key === ',') {
            event.preventDefault();
            commitDraft();
          }
          if (event.key === 'Backspace' && draft === '' && values.length > 0) {
            onChange(values.slice(0, -1));
          }
        }}
        onBlur={commitDraft}
      />
    </div>
  );
};

const FieldHint = ({ children }: { children: React.ReactNode }) => (
  <p className="text-xs leading-relaxed text-muted-foreground">{children}</p>
);

// Admin card managing the OIDC single sign-on of the deployment, rendered by
// the SSO page (Access & Security). Saving runs a live discovery against
// the issuer server-side: configuration mistakes surface immediately with
// the IdP's own error message.
export const SsoConfigCard = () => {
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const settingsQuery = useQuery({
    queryKey: ['ssoSettings'],
    queryFn: () => api.getSsoSettings(),
  });

  const [form, setForm] = useState<SsoFormState>(emptyForm);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [inlineError, setInlineError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [isTogglingEnabled, setIsTogglingEnabled] = useState(false);
  const [isRemoveDialogOpen, setIsRemoveDialogOpen] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);
  const [copied, setCopied] = useState(false);

  const settings = settingsQuery.data ?? null;
  // A failed fetch leaves data undefined, which must never read as "not
  // configured": the empty form is reserved for a successful 404 (null).
  // After a successful load, a failing refetch keeps showing the known state.
  const loadFailed = settingsQuery.isError && settingsQuery.data === undefined;

  const lastPopulatedSignature = useRef<string | null | undefined>(undefined);
  useEffect(() => {
    const signature = settingsFormSignature(settings);
    if (signature === lastPopulatedSignature.current) {
      return;
    }
    lastPopulatedSignature.current = signature;
    if (settings) {
      setForm(formFromSettings(settings));
      // Surface the advanced section when it holds non-default values,
      // otherwise the stored configuration would be partly invisible.
      if (settings.scopes !== DEFAULT_SCOPES || settings.groupsClaim !== DEFAULT_GROUPS_CLAIM) {
        setShowAdvanced(true);
      }
    } else {
      setForm(emptyForm);
    }
  }, [settings]);

  const redirectUri = settings?.redirectUri ?? api.ssoRedirectUri();
  const secretMissing = !!settings && !settings.hasClientSecret;

  const setField = <Key extends keyof SsoFormState>(field: Key, value: SsoFormState[Key]) =>
    setForm(current => ({ ...current, [field]: value }));

  const handleCopyRedirectUri = async () => {
    try {
      await navigator.clipboard.writeText(redirectUri);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast({
        title: 'Could not copy',
        description: 'Copy the redirect URI manually.',
        variant: 'destructive',
      });
    }
  };

  const handleSave = async () => {
    setInlineError(null);
    setIsSaving(true);
    try {
      const saved = await api.saveSsoSettings({
        issuer: form.issuer.trim(),
        clientId: form.clientId.trim(),
        clientSecret: form.clientSecret || undefined,
        providerName: form.providerName.trim(),
        scopes: form.scopes.trim(),
        // The toggle owns the enabled flag; saving the form preserves the
        // stored state, and the very first save goes live directly.
        enabled: settings?.enabled ?? true,
        allowedEmailDomains: form.allowedEmailDomains,
        allowedGroups: form.allowedGroups,
        groupsClaim: form.groupsClaim.trim(),
        trustUnverifiedEmail: form.trustUnverifiedEmail,
        manualUserValidation: form.manualUserValidation,
      });
      queryClient.setQueryData(['ssoSettings'], saved);
      queryClient.invalidateQueries({ queryKey: ['ssoPublicConfig'] });
      toast({
        title: 'SSO configuration saved',
        description: saved.enabled
          ? `OIDC discovery succeeded against ${saved.issuer}.`
          : 'Stored without testing the connection, since SSO is turned off.',
      });
    } catch (error) {
      setInlineError(describeApiError(error, 'Could not save the SSO configuration').description);
    } finally {
      setIsSaving(false);
    }
  };

  // The toggle acts on the stored configuration right away, with its own
  // request built from the stored values: unsaved form edits are neither
  // saved nor lost by flipping it. Disabling skips the server-side discovery
  // on purpose, so SSO can be turned off even while the IdP is down.
  const handleToggleEnabled = async (checked: boolean) => {
    if (!settings) {
      return;
    }
    setIsTogglingEnabled(true);
    try {
      const saved = await api.saveSsoSettings({
        issuer: settings.issuer,
        clientId: settings.clientId,
        clientSecret: undefined,
        providerName: settings.providerName,
        scopes: settings.scopes,
        enabled: checked,
        allowedEmailDomains: settings.allowedEmailDomains,
        allowedGroups: settings.allowedGroups,
        groupsClaim: settings.groupsClaim,
        trustUnverifiedEmail: settings.trustUnverifiedEmail,
        manualUserValidation: settings.manualUserValidation,
      });
      queryClient.setQueryData(['ssoSettings'], saved);
      queryClient.invalidateQueries({ queryKey: ['ssoPublicConfig'] });
      toast(
        saved.enabled
          ? {
              title: 'SSO sign-in enabled',
              description: `The connection was verified; the login page now offers "Continue with ${saved.providerName}".`,
            }
          : {
              title: 'SSO sign-in disabled',
              description:
                'The configuration is kept. Password sign-in is back for accounts that have a password.',
            }
      );
    } catch (error) {
      toast({
        ...describeApiError(
          error,
          checked ? 'Could not enable SSO sign-in' : 'Could not disable SSO sign-in'
        ),
        variant: 'destructive',
      });
    } finally {
      setIsTogglingEnabled(false);
    }
  };

  const handleRemove = async () => {
    setIsRemoving(true);
    try {
      await api.deleteSsoSettings();
      queryClient.setQueryData(['ssoSettings'], null);
      queryClient.invalidateQueries({ queryKey: ['ssoPublicConfig'] });
      setIsRemoveDialogOpen(false);
      setInlineError(null);
      toast({
        title: 'SSO configuration removed',
        description:
          'Password sign-in is back for accounts that have a password. Members provisioned by SSO cannot sign in until SSO is configured again.',
      });
    } catch (error) {
      toast({
        ...describeApiError(error, 'Could not remove the SSO configuration'),
        variant: 'destructive',
      });
    } finally {
      setIsRemoving(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex flex-wrap items-center gap-2">
          <KeyRound className="h-5 w-5 text-muted-foreground" strokeWidth={2} />
          Single sign-on (OIDC)
          {settingsQuery.isLoading || loadFailed ? null : settings ? (
            settings.enabled ? (
              <Badge>Active</Badge>
            ) : (
              <Badge variant="secondary">Configured, turned off</Badge>
            )
          ) : (
            <Badge variant="outline">Not configured</Badge>
          )}
        </CardTitle>
        <CardDescription>
          Let your team sign in through your identity provider: Microsoft Entra ID, Okta, Google
          Workspace, Keycloak or any OpenID Connect issuer. Accounts are created automatically as
          members on their first sign-in, and admins keep their password as a break-glass access.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {settingsQuery.isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-9 w-full" />
            <Skeleton className="h-9 w-2/3" />
            <Skeleton className="h-9 w-full" />
          </div>
        ) : loadFailed ? (
          <Alert variant="destructive">
            <TriangleAlert className="h-4 w-4" />
            <AlertTitle>Could not load the SSO configuration</AlertTitle>
            <AlertDescription className="flex flex-col items-start gap-3">
              <span className="break-words">
                {
                  describeApiError(settingsQuery.error, 'The server could not be reached.')
                    .description
                }
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => settingsQuery.refetch()}
                disabled={settingsQuery.isFetching}>
                {settingsQuery.isFetching ? 'Retrying…' : 'Retry'}
              </Button>
            </AlertDescription>
          </Alert>
        ) : (
          <div className="space-y-6">
            {secretMissing && (
              <Alert variant="destructive">
                <ShieldAlert className="h-4 w-4" />
                <AlertTitle>The stored client secret is no longer readable</AlertTitle>
                <AlertDescription>
                  This happens when the DB keys master key changes. SSO sign-ins fail until you
                  paste the client secret again and save.
                </AlertDescription>
              </Alert>
            )}

            {/* The switch only appears once a configuration exists: showing an
                "on" toggle over an empty form reads as "SSO is active" when
                nothing is configured yet. The first save enables SSO. It
                applies immediately (its own request against the stored
                configuration), independently of the form below. */}
            {settings && (
              <div className="flex items-start justify-between gap-4 rounded-lg border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label htmlFor="sso-enabled">Enable SSO sign-in</Label>
                  <FieldHint>
                    Applies immediately. While active, members must sign in through SSO and new
                    accounts are provisioned automatically; password sign-in stays available to
                    admins. Turning it off keeps the configuration.
                  </FieldHint>
                </div>
                <Switch
                  aria-label="Enable SSO sign-in"
                  checked={settings.enabled}
                  disabled={isTogglingEnabled || isSaving}
                  onCheckedChange={handleToggleEnabled}
                />
              </div>
            )}

            <div className="space-y-1.5">
              <Label htmlFor="sso-redirect-uri">Redirect URI</Label>
              <div className="flex items-center gap-2">
                <Input
                  id="sso-redirect-uri"
                  readOnly
                  value={redirectUri}
                  className="bg-muted/40 font-mono text-xs"
                  onFocus={event => event.target.select()}
                />
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  onClick={handleCopyRedirectUri}
                  title="Copy the redirect URI">
                  {copied ? (
                    <Check className="h-4 w-4 text-emerald-600" />
                  ) : (
                    <Copy className="h-4 w-4" />
                  )}
                </Button>
              </div>
              <FieldHint>
                Derived from BASE_URL. Register it as a web redirect URI in your identity provider's
                app registration before saving.
              </FieldHint>
            </div>

            <div className="grid gap-5 sm:grid-cols-2">
              <div className="space-y-1.5 sm:col-span-2">
                <Label htmlFor="sso-issuer">Issuer URL</Label>
                <Input
                  id="sso-issuer"
                  value={form.issuer}
                  onChange={event => setField('issuer', event.target.value)}
                  placeholder="https://login.microsoftonline.com/{tenant-id}/v2.0"
                  spellCheck={false}
                />
                <FieldHint>
                  The OpenID Connect issuer of your provider. Saving fetches{' '}
                  <code className="rounded bg-muted px-1 py-0.5 text-[11px]">
                    /.well-known/openid-configuration
                  </code>{' '}
                  from it and reports any error right here.
                </FieldHint>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="sso-client-id">Client ID</Label>
                <Input
                  id="sso-client-id"
                  value={form.clientId}
                  onChange={event => setField('clientId', event.target.value)}
                  placeholder="00000000-0000-0000-0000-000000000000"
                  spellCheck={false}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="sso-client-secret">Client secret</Label>
                <Input
                  id="sso-client-secret"
                  type="password"
                  value={form.clientSecret}
                  onChange={event => setField('clientSecret', event.target.value)}
                  placeholder={settings?.hasClientSecret ? '••••••••••••' : ''}
                  autoComplete="new-password"
                />
                <FieldHint>
                  {settings?.hasClientSecret
                    ? 'A secret is stored. Leave the field empty to keep it, or paste a new one to replace it.'
                    : 'The client secret issued by your identity provider.'}
                </FieldHint>
              </div>
              <div className="space-y-1.5 sm:col-span-2">
                <Label htmlFor="sso-provider-name">Button label</Label>
                <Input
                  id="sso-provider-name"
                  value={form.providerName}
                  onChange={event => setField('providerName', event.target.value)}
                  placeholder="Microsoft"
                  className="sm:max-w-xs"
                />
                <FieldHint>
                  Shown on the login page as "Continue with {form.providerName.trim() || 'SSO'}".
                </FieldHint>
              </div>
            </div>

            <div className="space-y-4 rounded-lg border p-4">
              <div>
                <p className="text-sm font-medium">Who can sign in</p>
                <FieldHint>
                  Leave both lists empty to allow every account your identity provider
                  authenticates. When set, an account must match every configured restriction.
                  Restricting who gets a token in the provider itself (Entra's "Assignment
                  required", Okta app assignments) remains the strongest control.
                </FieldHint>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="sso-domains">Allowed email domains</Label>
                <TagListInput
                  id="sso-domains"
                  values={form.allowedEmailDomains}
                  onChange={values => setField('allowedEmailDomains', values)}
                  placeholder="acme.com"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="sso-groups">Allowed groups</Label>
                <TagListInput
                  id="sso-groups"
                  values={form.allowedGroups}
                  onChange={values => setField('allowedGroups', values)}
                  placeholder="dashboard-users"
                />
                <FieldHint>
                  Matched against the "{form.groupsClaim.trim() || DEFAULT_GROUPS_CLAIM}" claim of
                  the id_token. Your provider must be configured to send it; Microsoft Entra ID
                  sends group object IDs (GUIDs), not names. Google does not put groups in tokens,
                  so use email domains there instead.
                </FieldHint>
              </div>

              <div className="flex items-start justify-between gap-4 border-t pt-4">
                <div className="space-y-0.5">
                  <Label htmlFor="sso-trust-unverified">Trust emails from this provider</Label>
                  <FieldHint>
                    By default an account is only matched by an email the provider marked as
                    verified (email_verified), which stops an attacker who set someone else's
                    address at the provider from taking over their account. Turn this on only for a
                    provider that does not send email_verified — notably Microsoft Entra ID — on a
                    single tenant where users cannot self-assign addresses (with "Assignment
                    required" set). Leave it off for a multi-tenant provider.
                  </FieldHint>
                </div>
                <Switch
                  aria-label="Trust emails from this provider"
                  checked={form.trustUnverifiedEmail}
                  onCheckedChange={checked => setField('trustUnverifiedEmail', checked)}
                />
              </div>

              <div className="flex items-start justify-between gap-4 border-t pt-4">
                <div className="space-y-0.5">
                  <Label htmlFor="sso-manual-validation">Require admin approval</Label>
                  <FieldHint>
                    New accounts are created on their first sign-in but cannot enter until an admin
                    approves them on the Users page, where they show up as pending. Use this when
                    your provider authenticates more people than should reach this dashboard and
                    does not send group claims to filter on, which is the case for Google Workspace.
                    Accounts that already exist keep their access.
                  </FieldHint>
                </div>
                <Switch
                  aria-label="Require admin approval"
                  checked={form.manualUserValidation}
                  onCheckedChange={checked => setField('manualUserValidation', checked)}
                />
              </div>
            </div>

            <div className="space-y-4">
              <button
                type="button"
                onClick={() => setShowAdvanced(open => !open)}
                className="flex items-center gap-1 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground">
                <ChevronRight
                  className={cn('h-4 w-4 transition-transform', showAdvanced && 'rotate-90')}
                />
                Advanced settings
              </button>
              {showAdvanced && (
                <div className="grid gap-5 sm:grid-cols-2">
                  <div className="space-y-1.5">
                    <Label htmlFor="sso-scopes">Scopes</Label>
                    <Input
                      id="sso-scopes"
                      value={form.scopes}
                      onChange={event => setField('scopes', event.target.value)}
                      placeholder={DEFAULT_SCOPES}
                      spellCheck={false}
                    />
                    <FieldHint>
                      Space-separated, must include "openid". Okta needs the extra "groups" scope
                      when you restrict by group.
                    </FieldHint>
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="sso-groups-claim">Groups claim</Label>
                    <Input
                      id="sso-groups-claim"
                      value={form.groupsClaim}
                      onChange={event => setField('groupsClaim', event.target.value)}
                      placeholder={DEFAULT_GROUPS_CLAIM}
                      spellCheck={false}
                    />
                    <FieldHint>
                      The id_token claim carrying group membership, "{DEFAULT_GROUPS_CLAIM}" for
                      most providers.
                    </FieldHint>
                  </div>
                </div>
              )}
            </div>

            {inlineError && (
              <Alert variant="destructive">
                <TriangleAlert className="h-4 w-4" />
                <AlertTitle>The configuration was not saved</AlertTitle>
                <AlertDescription className="break-words">{inlineError}</AlertDescription>
              </Alert>
            )}

            <div className="flex flex-wrap items-center justify-between gap-3">
              <FieldHint>
                {!settings
                  ? 'Saving verifies the issuer with a live OIDC discovery, then SSO sign-in goes live for your team.'
                  : settings.enabled
                    ? 'Saving verifies the issuer with a live OIDC discovery before anything is stored.'
                    : 'SSO is turned off: saving stores the configuration without testing the connection.'}
              </FieldHint>
              <div className="flex items-center gap-2">
                {settings && (
                  <Button
                    variant="outline"
                    onClick={() => setIsRemoveDialogOpen(true)}
                    disabled={isSaving || isTogglingEnabled}>
                    Remove
                  </Button>
                )}
                <Button
                  onClick={handleSave}
                  disabled={
                    isSaving || isTogglingEnabled || !form.issuer.trim() || !form.clientId.trim()
                  }>
                  {isSaving ? 'Testing connection…' : 'Save and test connection'}
                </Button>
              </div>
            </div>
          </div>
        )}
      </CardContent>

      <DeleteDialog
        isOpen={isRemoveDialogOpen}
        onClose={() => setIsRemoveDialogOpen(false)}
        onConfirm={handleRemove}
        isDeleting={isRemoving}
        title="Remove SSO configuration"
        resourceName={settings?.providerName}
        descriptionText="The OIDC configuration will be deleted and the SSO button disappears from the login page. Password sign-in comes back for accounts that have a password. Members provisioned by SSO keep their account but have no password, so they cannot sign in until SSO is configured again."
        confirmButtonText="Remove configuration"
        isDeletingButtonText="Removing…"
      />
    </Card>
  );
};
