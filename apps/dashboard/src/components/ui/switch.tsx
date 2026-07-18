import { cn } from '@/lib/utils';

type SwitchProps = {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
  'aria-label'?: string;
};

// Lightweight toggle switch. A plain button styled as a track + knob, so it
// carries no extra dependency; the on-state uses the emerald accent shared by
// the enterprise UI.
export const Switch = ({ checked, onCheckedChange, disabled, ...props }: SwitchProps) => (
  <button
    type="button"
    role="switch"
    aria-checked={checked}
    disabled={disabled}
    onClick={event => {
      event.stopPropagation();
      onCheckedChange(!checked);
    }}
    className={cn(
      'relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50',
      checked ? 'bg-emerald-600' : 'bg-input'
    )}
    {...props}>
    <span
      className={cn(
        'inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform',
        checked ? 'translate-x-4' : 'translate-x-0.5'
      )}
    />
  </button>
);
