"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Activity,
  Building2,
  Code2,
  LayoutDashboard,
  Plus,
  ShieldCheck,
  User,
  Users,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useSidebarState } from "./sidebar-state";

type Item = {
  label: string;
  href: string;
  icon: React.ElementType;
  quickAction?: { href: string; label: string };
};

export interface SidebarProps {
  canSeeProcesses: boolean;
  canSeeAdmin: boolean;
}

export function Sidebar({ canSeeProcesses, canSeeAdmin }: SidebarProps) {
  const pathname = usePathname();
  const { collapsed, mobileOpen, closeMobile } = useSidebarState();

  const RESOURCES: Item[] = [
    { label: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
    ...(canSeeProcesses
      ? [
          {
            label: "Processes",
            href: "/processes",
            icon: Activity,
            quickAction: { href: "/processes/new", label: "New process" },
          } as Item,
          { label: "API docs", href: "/api-docs", icon: Code2 },
        ]
      : []),
  ];

  const ACCOUNT: Item[] = [{ label: "Profile", href: "/profile", icon: User }];

  const ADMIN: Item[] = [
    { label: "Users", href: "/admin/users", icon: Users },
    { label: "Roles", href: "/admin/roles", icon: ShieldCheck },
    { label: "Organization", href: "/admin/organization", icon: Building2 },
  ];

  const isActive = (href: string) => pathname === href || pathname.startsWith(href + "/");

  return (
    <>
      {mobileOpen && (
        <button
          aria-label="Close sidebar"
          onClick={closeMobile}
          className="fixed inset-0 z-30 bg-background/70 backdrop-blur-sm md:hidden"
        />
      )}

      <aside
        className={cn(
          "h-full shrink-0 flex-col border-r border-sidebar-border bg-sidebar transition-[width,transform] duration-200 ease-out",
          "hidden md:flex",
          collapsed ? "md:w-16" : "md:w-60",
          mobileOpen
            ? "fixed inset-y-0 left-0 z-40 flex w-64 translate-x-0"
            : "max-md:-translate-x-full max-md:fixed max-md:inset-y-0 max-md:left-0 max-md:z-40 max-md:w-64",
        )}
      >
        <Link
          href="/dashboard"
          onClick={closeMobile}
          className={cn(
            "flex h-14 shrink-0 items-center gap-2 border-b border-sidebar-border hover:bg-sidebar-accent/40",
            collapsed ? "justify-center px-2" : "px-4",
          )}
        >
          <span
            aria-hidden
            className="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-primary-foreground font-bold text-xs"
          >
            P
          </span>
          {!collapsed && (
            <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              pm2-go
            </span>
          )}
        </Link>

        <nav className="flex-1 overflow-y-auto py-3">
          <SidebarGroup label="Resources" collapsed={collapsed}>
            {RESOURCES.map((item) => (
              <SidebarItem
                key={item.label}
                item={item}
                active={isActive(item.href)}
                collapsed={collapsed}
                onNavigate={closeMobile}
              />
            ))}
          </SidebarGroup>

          <SidebarGroup label="Account" collapsed={collapsed}>
            {ACCOUNT.map((item) => (
              <SidebarItem
                key={item.label}
                item={item}
                active={isActive(item.href)}
                collapsed={collapsed}
                onNavigate={closeMobile}
              />
            ))}
          </SidebarGroup>

          {canSeeAdmin && (
            <SidebarGroup label="Admin" collapsed={collapsed}>
              {ADMIN.map((item) => (
                <SidebarItem
                  key={item.label}
                  item={item}
                  active={isActive(item.href)}
                  collapsed={collapsed}
                  onNavigate={closeMobile}
                />
              ))}
            </SidebarGroup>
          )}
        </nav>
      </aside>
    </>
  );
}

function SidebarGroup({
  label,
  collapsed,
  children,
}: {
  label: string;
  collapsed?: boolean;
  children: React.ReactNode;
}) {
  return (
    <>
      {!collapsed && (
        <div className="mt-3 flex w-full items-center px-4 py-1.5 text-xs font-medium text-muted-foreground">
          {label}
        </div>
      )}
      <ul className={cn("space-y-px", collapsed ? "px-2 pt-2" : "px-2")}>{children}</ul>
    </>
  );
}

function SidebarItem({
  item,
  active,
  collapsed,
  onNavigate,
}: {
  item: Item;
  active?: boolean;
  collapsed?: boolean;
  onNavigate?: () => void;
}) {
  return (
    <li className="relative">
      <Link
        href={item.href}
        onClick={onNavigate}
        title={collapsed ? item.label : undefined}
        className={cn(
          "group flex w-full items-center rounded-md px-2 py-1.5 text-sm transition-colors",
          collapsed ? "justify-center" : "gap-2",
          !collapsed && item.quickAction && "pr-9",
          active
            ? "bg-sidebar-accent text-sidebar-accent-foreground"
            : "text-sidebar-foreground hover:bg-sidebar-accent/60 hover:text-foreground",
        )}
      >
        <item.icon className="h-4 w-4 shrink-0" />
        {!collapsed && <span className="flex-1 truncate">{item.label}</span>}
      </Link>
      {!collapsed && item.quickAction && (
        <Link
          href={item.quickAction.href}
          onClick={onNavigate}
          aria-label={item.quickAction.label}
          title={item.quickAction.label}
          className="absolute right-1.5 top-1/2 inline-flex h-6 w-6 -translate-y-1/2 items-center justify-center rounded-md border border-border bg-background text-muted-foreground transition-colors hover:border-foreground/40 hover:bg-sidebar-accent/60 hover:text-foreground"
        >
          <Plus className="h-3.5 w-3.5" />
        </Link>
      )}
    </li>
  );
}
