import { Button } from '@/components/ui/button';
import { GitBranch, Pencil, Radio } from 'lucide-react';
import { Link } from 'react-router';

type BranchStatus = {
  label: 'Current update' | 'Current rollout';
  commitHash: string;
  percentage?: number;
};

type Props = {
  branchName?: string | null;
  channelNames: string[];
  focus?: 'channel' | 'branch';
  branchStatus?: BranchStatus;
  rolloutStatus?: BranchStatus;
  rollout?: {
    branchName: string;
    percentage: number;
  };
  onEdit?: () => void;
};

const MappingNode = ({
  type,
  name,
  to,
  active,
  meta,
  status,
  rollout,
  branchFocused,
}: {
  type: 'channel' | 'branch';
  name: string;
  to?: string;
  active?: boolean;
  meta?: string;
  status?: Props['branchStatus'];
  rollout?: boolean;
  branchFocused?: boolean;
}) => {
  const Icon = type === 'channel' ? Radio : GitBranch;
  const content = (
    <div
      className={`flex min-h-[76px] min-w-0 items-center gap-3 rounded-lg border px-5 py-4 shadow-card transition-all duration-200 group-hover:-translate-y-0.5 group-hover:border-input motion-reduce:transition-none motion-reduce:group-hover:translate-y-0 ${
        active
          ? 'border-primary/35 bg-primary/10 text-foreground'
          : rollout
            ? 'border-emerald-400/30 bg-emerald-400/[0.07] text-foreground'
            : 'bg-secondary text-foreground'
      }`}>
      <Icon
        className={`h-5 w-5 shrink-0 ${
          active
            ? 'text-primary'
            : rollout
              ? 'text-emerald-700 dark:text-emerald-300'
              : 'text-muted-foreground'
        }`}
      />
      <div className="min-w-0">
        <p className="text-xs font-medium text-muted-foreground">
          {rollout
            ? 'Rollout branch'
            : type === 'channel'
              ? 'Channel'
              : branchFocused
                ? 'Branch'
                : 'Default branch'}
        </p>
        <p className="truncate text-sm font-semibold text-foreground">{name}</p>
        {meta && <p className="mt-0.5 text-xs text-muted-foreground">{meta}</p>}
        {status && (
          <div className="mt-1 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
            <span>{status.label}</span>
            {status.percentage != null && (
              <span className="rounded bg-emerald-400/10 px-1.5 py-0.5 font-medium text-emerald-700 dark:text-emerald-300">
                {status.percentage}%
              </span>
            )}
            <span className="font-medium text-foreground">{status.commitHash}</span>
          </div>
        )}
      </div>
    </div>
  );
  return to ? (
    <Link
      to={to}
      className="group min-w-0 rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/30">
      {content}
    </Link>
  ) : (
    content
  );
};

const Connector = ({ split = false }: { split?: boolean }) => (
  <div className="relative hidden h-full min-h-[76px] md:block">
    <span
      className={`absolute left-0 top-1/2 border-t border-muted-foreground/40 ${
        split ? 'w-1/2' : 'w-full'
      }`}
    />
    {split ? (
      <>
        <span className="absolute bottom-[23%] left-1/2 top-[23%] border-l border-muted-foreground/40" />
        <span className="absolute left-1/2 right-0 top-[23%] border-t border-muted-foreground/40" />
        <span className="absolute bottom-[23%] left-1/2 right-0 border-t border-muted-foreground/40" />
        <span className="absolute -right-1 top-[calc(23%-3px)] h-2 w-2 rounded-full border border-muted-foreground bg-card" />
        <span className="absolute -right-1 bottom-[calc(23%-3px)] h-2 w-2 rounded-full border border-muted-foreground bg-card" />
      </>
    ) : (
      <span className="absolute -right-1 top-[calc(50%-3px)] h-2 w-2 rounded-full border border-muted-foreground bg-card" />
    )}
    <span className="absolute -left-1 top-[calc(50%-3px)] h-2 w-2 rounded-full border border-muted-foreground bg-card" />
  </div>
);

export const ChannelBranchMapping = ({
  branchName,
  channelNames,
  focus = 'channel',
  branchStatus,
  rolloutStatus,
  rollout,
  onEdit,
}: Props) => {
  const mappings = channelNames.length > 0 ? channelNames : [null];
  const channelDetailMapping = mappings.length === 1 && !!mappings[0] && !!rollout;
  return (
    <section className="overflow-hidden rounded-lg border bg-card shadow-card">
      <header className="flex min-h-14 items-center justify-between gap-4 border-b px-5">
        <div className="flex items-center gap-2">
          <GitBranch className="h-4 w-4 text-muted-foreground" />
          <h2 className="text-sm font-semibold">Channel-branch mapping</h2>
        </div>
        {onEdit && (
          <Button variant="secondary" size="sm" onClick={onEdit}>
            <Pencil className="h-3.5 w-3.5" />
            Edit
          </Button>
        )}
      </header>
      <div
        className="min-h-[300px] space-y-4 px-8 py-10 lg:px-12 lg:py-12"
        style={{
          backgroundImage: 'radial-gradient(hsl(var(--graph-dot) / 0.45) 1px, transparent 1px)',
          backgroundSize: '14px 14px',
        }}>
        {channelDetailMapping ? (
          <div className="mx-auto grid max-w-[1480px] items-center gap-4 md:grid-cols-[minmax(0,1fr)_112px_minmax(0,1fr)] md:gap-0">
            <MappingNode
              type="channel"
              name={mappings[0] as string}
              to={`/channels/${encodeURIComponent(mappings[0] as string)}`}
              active={focus === 'channel'}
            />
            <Connector split />
            <div className="grid gap-5">
              {branchName ? (
                <MappingNode
                  type="branch"
                  name={branchName}
                  to={`/branches/${encodeURIComponent(branchName)}`}
                  meta={`${100 - rollout.percentage}% of devices`}
                  active={focus === 'branch'}
                  status={branchStatus}
                  branchFocused={focus === 'branch'}
                />
              ) : (
                <div className="rounded-lg border border-dashed bg-secondary px-4 py-5 text-sm text-muted-foreground">
                  No default branch mapped
                </div>
              )}
              <MappingNode
                type="branch"
                name={rollout.branchName}
                to={`/branches/${encodeURIComponent(rollout.branchName)}`}
                meta={`${rollout.percentage}% of devices`}
                status={rolloutStatus}
                rollout
              />
            </div>
          </div>
        ) : (
          mappings.map((channelName, index) => (
            <div
              key={channelName ?? `unmapped-${index}`}
              className="mx-auto grid max-w-[1480px] items-center gap-4 md:grid-cols-[minmax(0,1fr)_112px_minmax(0,1fr)] md:gap-0">
              {channelName ? (
                <MappingNode
                  type="channel"
                  name={channelName}
                  to={`/channels/${encodeURIComponent(channelName)}`}
                  active={focus === 'channel'}
                />
              ) : (
                <div className="rounded-lg border border-dashed bg-secondary px-4 py-5 text-sm text-muted-foreground">
                  No channel mapped
                </div>
              )}
              <Connector />
              {branchName ? (
                <MappingNode
                  type="branch"
                  name={branchName}
                  to={`/branches/${encodeURIComponent(branchName)}`}
                  active={focus === 'branch'}
                  status={branchStatus}
                  branchFocused={focus === 'branch'}
                />
              ) : (
                <div className="rounded-lg border border-dashed bg-secondary px-4 py-5 text-sm text-muted-foreground">
                  No branch mapped
                </div>
              )}
            </div>
          ))
        )}
      </div>
    </section>
  );
};
