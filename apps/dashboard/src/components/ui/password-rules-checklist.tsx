import { Check, X } from 'lucide-react';
import { cn } from '@/lib/utils';
import { PASSWORD_RULES } from '@/lib/password-policy';

// Live feedback for the password policy: one line per rule, ticking off as the
// user types. Rendered next to every password-choosing field so a policy 400
// from the server is a bug, not a surprise.
export const PasswordRulesChecklist = ({ password }: { password: string }) => (
  <ul className="space-y-1 rounded-lg border bg-muted/30 p-3">
    {PASSWORD_RULES.map(rule => {
      const passed = rule.test(password);
      return (
        <li
          key={rule.id}
          className={cn(
            'flex items-center gap-2 text-xs',
            passed ? 'text-foreground' : 'text-muted-foreground'
          )}>
          {passed ? (
            <Check className="h-3.5 w-3.5 text-primary" strokeWidth={2.5} />
          ) : (
            <X className="h-3.5 w-3.5 text-muted-foreground/60" strokeWidth={2} />
          )}
          {rule.label}
        </li>
      );
    })}
  </ul>
);
