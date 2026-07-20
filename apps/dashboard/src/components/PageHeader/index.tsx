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
  <header className="mb-8 space-y-1.5 border-b pb-6">
    <div className="flex items-center justify-between gap-4">
      <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
      {actions}
    </div>
    {description && (
      <div className="max-w-3xl text-sm leading-relaxed text-muted-foreground">{description}</div>
    )}
  </header>
);
