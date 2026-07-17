import { Radio } from 'lucide-react';
import { Input } from '@/components/ui/input.tsx';
import { Button } from '@/components/ui/button.tsx';
import { Label } from '@/components/ui/label.tsx';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { Form, FormControl, FormField, FormItem, FormMessage } from '@/components/ui/form.tsx';
import { useCallback } from 'react';
import { setTokens } from '@/lib/auth.ts';
import { useNavigate } from 'react-router';
import { api, ApiProblemError } from '@/lib/api.ts';

const FormSchema = z.object({
  email: z.string().email({
    message: 'Enter a valid email address',
  }),
  password: z.string().min(1, {
    message: 'Enter your password',
  }),
});

export const Login = () => {
  const form = useForm<z.infer<typeof FormSchema>>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      email: '',
      password: '',
    },
  });
  const navigate = useNavigate();

  const onSubmit = useCallback(
    async (data: z.infer<typeof FormSchema>) => {
      try {
        const response = await api.login(data.email, data.password);
        setTokens(response.token, response.refreshToken);
        navigate('/');
      } catch (error) {
        // Only an actual 401 means wrong credentials. A misconfigured server
        // (e.g. ADMIN_EMAIL not set in stateless mode) answers with an
        // actionable detail, and a fetch that never reached the server must
        // not masquerade as a credentials problem.
        let message = 'Could not reach the server. Check that it is running.';
        if (error instanceof ApiProblemError) {
          message = error.status === 401 ? 'Invalid email or password' : error.detail;
        }
        form.setError('password', {
          type: 'server',
          message,
        });
      }
    },
    [form, navigate]
  );

  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-muted/50 px-4">
      <div className="w-full max-w-sm">
        <div className="rounded-2xl border bg-background p-8 shadow-elevated">
          <div className="mb-8 flex flex-col items-center gap-3 text-center">
            <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-primary text-primary-foreground">
              <Radio className="h-5 w-5" strokeWidth={2} />
            </div>
            <div className="space-y-1">
              <h1 className="text-lg font-semibold tracking-tight">Expo Open OTA</h1>
              <p className="text-sm text-muted-foreground">
                Sign in to manage your over-the-air updates
              </p>
            </div>
          </div>

          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-5">
              <FormField
                control={form.control}
                name="email"
                render={({ field, fieldState }) => {
                  return (
                    <FormItem className="space-y-1.5">
                      <Label htmlFor="login-email">Email</Label>
                      <FormControl>
                        <Input
                          id="login-email"
                          type="email"
                          autoFocus
                          autoComplete="username"
                          {...field}
                        />
                      </FormControl>
                      <FormMessage>{fieldState.error?.message}</FormMessage>
                    </FormItem>
                  );
                }}
              />
              <FormField
                control={form.control}
                name="password"
                render={({ field, fieldState }) => {
                  return (
                    <FormItem className="space-y-1.5">
                      <Label htmlFor="login-password">Password</Label>
                      <FormControl>
                        <Input
                          id="login-password"
                          type="password"
                          autoComplete="current-password"
                          {...field}
                        />
                      </FormControl>
                      <FormMessage>{fieldState.error?.message}</FormMessage>
                    </FormItem>
                  );
                }}
              />
              <Button type="submit" className="w-full" disabled={form.formState.isSubmitting}>
                {form.formState.isSubmitting ? 'Signing in…' : 'Sign in'}
              </Button>
            </form>
          </Form>
        </div>

        <p className="mt-6 text-center text-xs text-muted-foreground">
          Self-hosted OTA updates for Expo apps
        </p>
      </div>
    </div>
  );
};
