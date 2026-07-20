import { ReactNode } from 'react';
import { Lock } from 'lucide-react';

// Shown to non-admin users where admin-only actions would otherwise appear, so
// the absence of a button reads as "you need an admin" rather than a bug.
export const AdminOnlyNote = ({ children }: { children: ReactNode }) => (
  <div className="flex items-center gap-2.5 rounded-xl border border-dashed bg-muted/30 px-4 py-3 text-xs text-muted-foreground">
    <Lock className="h-3.5 w-3.5 shrink-0" strokeWidth={1.75} />
    <span>{children}</span>
  </div>
);
