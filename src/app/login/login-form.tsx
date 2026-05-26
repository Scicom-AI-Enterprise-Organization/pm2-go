"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { signIn } from "next-auth/react";
import {
  AlertCircle,
  Building2,
  Check,
  Eye,
  EyeOff,
  KeyRound,
  Loader2,
  Lock,
  Mail,
  ShieldCheck,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { GoogleIcon } from "@/components/auth/provider-icons";

interface LoginFormProps {
  callbackUrl: string;
  error?: string;
  providers: {
    azure: boolean;
    google: boolean;
    keycloak: boolean;
    saml: boolean;
  };
}

const ERROR_MESSAGES: Record<string, string> = {
  CredentialsSignin: "Invalid email or password.",
  OAuthSignin: "Could not start SSO sign-in. Please try again.",
  OAuthCallback: "SSO callback failed. Please try again.",
  OAuthAccountNotLinked: "This email is already linked to another sign-in method.",
  Callback: "Authentication callback failed. Please try again.",
  AccessDenied: "Access denied.",
  SessionRequired: "Please sign in to continue.",
};

export function LoginForm({ callbackUrl, error: initialError, providers }: LoginFormProps) {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [rememberMe, setRememberMe] = useState(true);
  const [error, setError] = useState(
    initialError ? ERROR_MESSAGES[initialError] ?? `Sign in failed: ${initialError}` : "",
  );
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await signIn("credentials", {
        email: email.toLowerCase().trim(),
        password,
        redirect: false,
      });
      if (!res) {
        setError("No response from server.");
        return;
      }
      if (res.error) {
        setError("Invalid email or password.");
        return;
      }
      router.replace(callbackUrl);
      router.refresh();
    } finally {
      setLoading(false);
    }
  }

  const ssoButtons = [
    providers.google && {
      key: "google",
      label: "Google",
      icon: <GoogleIcon className="h-4 w-4" />,
      onClick: () => signIn("google", { callbackUrl }),
    },
    providers.azure && {
      key: "azure",
      label: "Azure AD",
      icon: <KeyRound className="h-4 w-4" />,
      onClick: () => signIn("microsoft-entra-id", { callbackUrl }),
    },
    providers.keycloak && {
      key: "keycloak",
      label: "Keycloak",
      icon: <Building2 className="h-4 w-4" />,
      onClick: () => signIn("keycloak", { callbackUrl }),
    },
    providers.saml && {
      key: "saml",
      label: "SAML SSO",
      icon: <ShieldCheck className="h-4 w-4" />,
      onClick: () => (window.location.href = "/api/auth/saml/login"),
    },
  ].filter(Boolean) as { key: string; label: string; icon: React.ReactNode; onClick: () => void }[];

  return (
    <>
      <div className="mb-6 flex justify-center">
        <div className="flex size-12 items-center justify-center rounded-xl bg-primary/10">
          <Lock className="size-6 text-primary" />
        </div>
      </div>

      <div className="mb-6 text-center">
        <h1 className="text-2xl font-bold text-foreground">Welcome back</h1>
        <p className="mt-1 text-sm text-muted-foreground">Sign in to manage your processes</p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        {error && (
          <div className="flex items-center gap-2 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
            <AlertCircle className="size-4 shrink-0" />
            {error}
          </div>
        )}

        <div className="space-y-1.5">
          <label htmlFor="email" className="text-sm font-medium text-foreground">
            Email
          </label>
          <div className="flex h-11 items-center gap-2 rounded-lg border border-transparent bg-slate-100 px-3 transition-colors focus-within:border-primary focus-within:ring-2 focus-within:ring-primary/20 dark:bg-slate-900">
            <Mail className="size-4 shrink-0 text-muted-foreground" />
            <input
              id="email"
              type="email"
              autoComplete="email"
              placeholder="name@company.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              className="autofill-fix flex-1 border-0 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
            />
          </div>
        </div>

        <div className="space-y-1.5">
          <label htmlFor="password" className="text-sm font-medium text-foreground">
            Password
          </label>
          <div className="flex h-11 items-center gap-2 rounded-lg border border-transparent bg-slate-100 px-3 transition-colors focus-within:border-primary focus-within:ring-2 focus-within:ring-primary/20 dark:bg-slate-900">
            <Lock className="size-4 shrink-0 text-muted-foreground" />
            <input
              id="password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              className="autofill-fix flex-1 border-0 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="text-muted-foreground transition-colors hover:text-foreground"
              aria-label={showPassword ? "Hide password" : "Show password"}
            >
              {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
            </button>
          </div>
        </div>

        <div className="flex items-center justify-between">
          <label className="flex cursor-pointer items-center gap-2 text-sm">
            <button
              type="button"
              onClick={() => setRememberMe(!rememberMe)}
              className={`flex size-4 items-center justify-center rounded border transition-colors ${
                rememberMe
                  ? "border-primary bg-primary text-primary-foreground"
                  : "border-border bg-background"
              }`}
              aria-label="Remember me"
            >
              {rememberMe && <Check className="size-3" />}
            </button>
            <span className="text-muted-foreground">Remember me</span>
          </label>
        </div>

        <Button type="submit" className="h-11 w-full" disabled={loading}>
          {loading && <Loader2 className="size-4 animate-spin" />}
          {loading ? "Signing in..." : "Sign in"}
        </Button>
      </form>

      {ssoButtons.length > 0 && (
        <>
          <div className="my-5 flex items-center gap-3">
            <div className="h-px flex-1 bg-border" />
            <span className="text-xs text-muted-foreground">or continue with</span>
            <div className="h-px flex-1 bg-border" />
          </div>
          <div className="grid grid-cols-2 gap-2">
            {ssoButtons.map((b) => (
              <Button
                key={b.key}
                type="button"
                variant="outline"
                className="h-11"
                onClick={b.onClick}
              >
                {b.icon}
                {b.label}
              </Button>
            ))}
          </div>
        </>
      )}
    </>
  );
}
