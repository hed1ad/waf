"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Shield, Activity, List, Settings, BarChart3, Globe } from "lucide-react";
import { cn } from "@/lib/utils";

const nav = [
  { href: "/",        label: "Overview",  icon: BarChart3 },
  { href: "/events",  label: "Events",    icon: Activity  },
  { href: "/rules",   label: "Rules",     icon: List      },
  { href: "/geo",     label: "GeoMap",    icon: Globe     },
  { href: "/settings",label: "Settings",  icon: Settings  },
];

export function Sidebar() {
  const pathname = usePathname();
  return (
    <aside className="flex w-56 flex-col border-r border-border bg-card">
      <div className="flex items-center gap-2 px-4 py-5 border-b border-border">
        <Shield className="h-6 w-6 text-primary" />
        <span className="font-semibold text-sm tracking-wide">WAF Shield</span>
      </div>
      <nav className="flex-1 py-4">
        {nav.map(({ href, label, icon: Icon }) => (
          <Link
            key={href}
            href={href}
            className={cn(
              "flex items-center gap-3 px-4 py-2.5 text-sm transition-colors",
              pathname === href
                ? "bg-primary/10 text-primary font-medium"
                : "text-muted-foreground hover:bg-muted hover:text-foreground"
            )}
          >
            <Icon className="h-4 w-4" />
            {label}
          </Link>
        ))}
      </nav>
      <div className="px-4 py-3 border-t border-border">
        <p className="text-xs text-muted-foreground">ModSecurity v3 • CRS v4</p>
      </div>
    </aside>
  );
}
