import { redirect } from "next/navigation";
import Link from "next/link";
import { auth } from "@/lib/auth";
import { LoginForm } from "./login-form";

export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ callbackUrl?: string; error?: string }>;
}) {
  const session = await auth();
  const params = await searchParams;
  if (session?.user) redirect(params.callbackUrl ?? "/dashboard");

  const enabledProviders = {
    azure: !!process.env.AUTH_AZURE_AD_CLIENT_ID,
    google: !!process.env.AUTH_GOOGLE_CLIENT_ID,
    keycloak: !!process.env.AUTH_KEYCLOAK_CLIENT_ID,
    saml: !!process.env.AUTH_SAML_ENTRY_POINT,
  };

  return (
    <div className="grid min-h-screen w-full lg:grid-cols-2">
      {/* Hero (hidden on mobile) */}
      <div className="relative hidden overflow-hidden bg-gradient-to-br from-primary/15 via-background to-primary/5 lg:block">
        <div
          aria-hidden
          className="absolute inset-0 opacity-[0.4] [background-image:radial-gradient(circle,_var(--muted-foreground)_1px,_transparent_1px)] [background-size:18px_18px]"
        />
        <div className="relative z-10 flex h-full flex-col justify-between p-10">
          <Link href="/" className="inline-flex items-center gap-3 self-start">
            <span
              aria-hidden
              className="flex h-9 w-9 items-center justify-center rounded-md bg-primary text-primary-foreground font-bold"
            >
              P
            </span>
            <span className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
              pm2-go
            </span>
          </Link>
          <div className="max-w-md space-y-3">
            <h2 className="text-3xl font-semibold tracking-tight text-foreground">
              Process management without babysitting.
            </h2>
            <p className="text-sm text-muted-foreground">
              Run, supervise, and tail your Node, Python, and binary services — all from a
              single Go daemon and a browser pane of glass.
            </p>
          </div>
        </div>
      </div>

      {/* Form */}
      <div className="flex flex-col items-center justify-center bg-slate-50 px-6 py-10 dark:bg-slate-950 sm:px-10">
        <div className="w-full max-w-md">
          <Link href="/" className="mb-6 inline-flex items-center gap-2 lg:hidden">
            <span
              aria-hidden
              className="flex h-8 w-8 items-center justify-center rounded-md bg-primary text-primary-foreground font-bold text-sm"
            >
              P
            </span>
            <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              pm2-go
            </span>
          </Link>
          <LoginForm
            callbackUrl={params.callbackUrl ?? "/dashboard"}
            error={params.error}
            providers={enabledProviders}
          />
        </div>
      </div>
    </div>
  );
}
