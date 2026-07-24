// Shared page heading so every view opens with the same rhythm: title on the
// left, optional actions on the right, optional description underneath.
export const PageHeader = ({
  title,
  description,
  actions,
}: {
  title: React.ReactNode;
  description?: React.ReactNode;
  actions?: React.ReactNode;
}) => (
  <header className="mb-7 space-y-2 border-b border-border/80 pb-6">
    <div className="flex items-center justify-between gap-4">
      <h1 className="font-display text-[28px] font-semibold tracking-tight text-foreground">
        {title}
      </h1>
      {actions}
    </div>
    {description && (
      <div className="max-w-3xl text-sm leading-relaxed text-muted-foreground">{description}</div>
    )}
  </header>
);
