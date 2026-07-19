import { Progress } from '@/components/ui/progress';
import { cn } from '@/lib/utils';

// Read-only visual of a rollout percentage: a slim emerald bar plus its label.
// Reused in the channel table cell, the per-update card and the details sheet.
export const RolloutBar = ({ value, className }: { value: number; className?: string }) => (
  <span className={cn('inline-flex items-center gap-2', className)}>
    <Progress
      value={value}
      className="h-1.5 w-24 bg-emerald-100"
      indicatorClassName="bg-emerald-500"
    />
    <span className="text-xs font-medium tabular-nums text-muted-foreground">{value}%</span>
  </span>
);
