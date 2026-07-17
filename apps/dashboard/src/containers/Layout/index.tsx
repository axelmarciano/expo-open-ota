import { AppSidebar } from '@/components/app-sidebar';

export const Layout = ({ children }: { children: React.ReactNode }) => {
  return (
    <div className="flex min-h-screen w-full">
      <AppSidebar />
      <main className="min-w-0 flex-1">
        <div className="mx-auto max-w-5xl px-8 py-10">{children}</div>
      </main>
    </div>
  );
};
