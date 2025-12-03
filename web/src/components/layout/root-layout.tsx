import { Outlet } from "react-router"
import { Sidebar } from "./sidebar"
import { Breadcrumb } from "./breadcrumb"
import { useState } from "react"

export function RootLayout() {
    const [, setIsSidebarCollapsed] = useState(false)

    return (
        <div className="flex h-screen w-full">
            <Sidebar
                displayConfig={{
                    monitor: true,
                    logs: true,
                    rules: true,
                    settings: true,
                }}
                onSidebarToggle={setIsSidebarCollapsed}
            />
            {/* Просто flex-1 без margin */}
            <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
                <Breadcrumb />
                <div className="flex-1 overflow-auto bg-gradient-to-br from-slate-50 to-white scrollbar-none">
                    <Outlet />
                </div>
            </div>
        </div>
    )
}