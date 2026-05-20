import clsx from "clsx";
import { ShieldAlert } from "lucide-react";
import { NavLink } from "react-router-dom";
import { navGroups } from "../routes";

export function Sidebar() {
  return (
    <aside className="sticky top-0 z-20 flex items-center gap-3 overflow-x-auto bg-slate-950 px-3 py-2 text-white lg:min-h-screen lg:flex-col lg:items-stretch lg:gap-6 lg:px-4 lg:py-5">
      <div className="flex items-center gap-3 rounded-md px-2 py-2">
        <ShieldAlert className="h-6 w-6 shrink-0 text-blue-300" />
        <div className="hidden lg:block">
          <div className="text-sm font-semibold">Sentry Lite</div>
          <div className="text-xs text-slate-400">错误监控后台</div>
        </div>
      </div>
      <nav className="flex gap-2 lg:grid">
        {navGroups.map((group) => (
          <div key={group.title} className="flex gap-1 lg:grid">
            <div className="hidden px-3 pt-2 text-[11px] font-semibold uppercase text-slate-500 lg:block">{group.title}</div>
            {group.items.map((item) => {
              const Icon = item.icon;
              return (
                <NavLink
                  key={item.path}
                  to={item.path}
                  className={({ isActive }) =>
                    clsx(
                      "flex h-10 items-center gap-2 rounded-md px-3 text-sm transition",
                      isActive ? "bg-slate-800 text-white" : "text-slate-300 hover:bg-slate-900 hover:text-white",
                    )
                  }
                  title={item.label}
                >
                  <Icon className="h-4 w-4 shrink-0" />
                  <span className="hidden lg:inline">{item.label}</span>
                </NavLink>
              );
            })}
          </div>
        ))}
      </nav>
    </aside>
  );
}
