import { useEffect, useState } from 'react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

const QUICK_PICKS = [1, 5, 10, 25, 50, 75];

// Number input for a rollout percentage plus quick-pick chips. Keeps a local
// string so an in-progress edit (empty field, a leading digit) is not clamped
// on every keystroke; a valid in-range value still propagates immediately so
// the parent's save button never reads a stale number, and blur normalizes.
export const PercentInput = ({
  value,
  onChange,
  min = 1,
  max = 99,
  disabled,
}: {
  value: number;
  onChange: (value: number) => void;
  min?: number;
  max?: number;
  disabled?: boolean;
}) => {
  const [text, setText] = useState(String(value));

  useEffect(() => {
    setText(String(value));
  }, [value]);

  const handleChange = (raw: string) => {
    setText(raw);
    const parsed = parseInt(raw, 10);
    if (!Number.isNaN(parsed) && parsed >= min && parsed <= max) {
      onChange(parsed);
    }
  };

  const handleBlur = () => {
    const parsed = parseInt(text, 10);
    const clamped = Number.isNaN(parsed) ? value : Math.min(max, Math.max(min, parsed));
    setText(String(clamped));
    onChange(clamped);
  };

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Input
          type="number"
          min={min}
          max={max}
          value={text}
          disabled={disabled}
          onChange={e => handleChange(e.target.value)}
          onBlur={handleBlur}
          className="w-20"
        />
        <span className="text-sm text-muted-foreground">% of devices</span>
      </div>
      <div className="flex flex-wrap gap-1.5">
        {QUICK_PICKS.filter(p => p >= min && p <= max).map(p => (
          <Button
            key={p}
            type="button"
            variant="outline"
            size="sm"
            disabled={disabled}
            onClick={() => handleChange(String(p))}
            className={cn(
              value === p &&
                'border-emerald-400/40 bg-emerald-400/10 text-emerald-700 hover:bg-emerald-400/15 hover:text-emerald-800 dark:text-emerald-300 dark:hover:text-emerald-200'
            )}>
            {p}%
          </Button>
        ))}
      </div>
    </div>
  );
};
