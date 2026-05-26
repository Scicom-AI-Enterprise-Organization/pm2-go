"use client";

import {
  LayoutDashboard,
  Users,
  ShieldCheck,
  Building2,
  UserCog,
  Sparkles,
  Activity,
} from "lucide-react";
import { PageSidebar } from "@/components/page-sidebar";

export function AppSidebar({
  isAdmin,
  canSeeProcesses = false,
}: {
  isAdmin: boolean;
  canSeeProcesses?: boolean;
}) {
  const sections = [
    {
      items: [
        { label: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
        { label: "AI Workspace", href: "/ai", icon: Sparkles },
        { label: "Profile", href: "/profile", icon: UserCog },
      ],
    },
    ...(canSeeProcesses
      ? [
          {
            title: "Operations",
            items: [{ label: "Processes", href: "/processes", icon: Activity }],
          },
        ]
      : []),
    ...(isAdmin
      ? [
          {
            title: "Admin",
            items: [
              { label: "Users", href: "/admin/users", icon: Users },
              { label: "Roles", href: "/admin/roles", icon: ShieldCheck },
              { label: "Organization", href: "/admin/organization", icon: Building2 },
            ],
          },
        ]
      : []),
  ];

  return <PageSidebar sections={sections} />;
}
