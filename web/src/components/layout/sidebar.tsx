"use client"

import { Link, useLocation, useNavigate } from "react-router"
import { cn } from "@/lib/utils"
import { Settings, Shield, BarChart2, FileText, LogOut, ChevronLeft, ChevronRight } from "lucide-react"
import { ROUTES } from "@/routes/constants"
import { useTranslation } from "react-i18next"
import type { TFunction } from "i18next"
import { useAuthStore } from "@/store/auth"
import { useState } from "react"

// Create sidebar config with display options
function createSidebarConfig(t: TFunction) {
    return [
        {
            title: t("sidebar.monitor"),
            icon: BarChart2,
            href: ROUTES.MONITOR,
            display: true,
        },
        {
            title: t("sidebar.logs"),
            icon: FileText,
            href: ROUTES.LOGS,
            display: true,
        },
        {
            title: t("sidebar.rules"),
            icon: Shield,
            href: ROUTES.RULES,
            display: true,
        },
        {
            title: t("sidebar.settings"),
            icon: Settings,
            href: ROUTES.SETTINGS,
            display: true,
        },
    ] as const
}

interface SidebarDisplayConfig {
    monitor?: boolean
    logs?: boolean
    rules?: boolean
    settings?: boolean
}

interface SidebarProps {
    displayConfig?: SidebarDisplayConfig
    onSidebarToggle?: (isCollapsed: boolean) => void
}

export function Sidebar({ displayConfig = {}, onSidebarToggle }: SidebarProps) {
    const location = useLocation()
    const { t } = useTranslation()
    const navigate = useNavigate()
    const { logout } = useAuthStore()
    const [isLogoutActive, setIsLogoutActive] = useState(false)
    const [isCollapsed, setIsCollapsed] = useState(false)

    // Get current first level path
    const currentFirstLevelPath = "/" + location.pathname.split("/")[1]

    // Generate sidebar items with display config
    const sidebarItems = createSidebarConfig(t).map((item) => {
        // Determine which config property based on path
        let configKey: keyof SidebarDisplayConfig = "monitor"
        if (item.href === ROUTES.LOGS) configKey = "logs"
        if (item.href === ROUTES.RULES) configKey = "rules"
        if (item.href === ROUTES.SETTINGS) configKey = "settings"

        // Use config value or default
        const shouldDisplay = displayConfig[configKey] !== undefined ? displayConfig[configKey] : item.display

        return {
            ...item,
            display: shouldDisplay,
        }
    })

    const handleLogout = () => {
        setIsLogoutActive(true)

        // Visual feedback before actual logout
        setTimeout(() => {
            logout()
            navigate("/login")
        }, 300)
    }

    const toggleSidebar = () => {
        const newState = !isCollapsed
        setIsCollapsed(newState)
        onSidebarToggle?.(newState)
    }

    return (
        <div
            className={cn(
                "text-white flex flex-col border-r border-slate-200 dark:border-none relative overflow-hidden transition-all duration-300 bg-sidebar-gradient flex-shrink-0", // добавлен flex-shrink-0
                isCollapsed ? "w-16" : "w-64"
            )}
        >
            {/* 霓虹灯效果 暗色模式 */}
            <div className="absolute inset-0 dark:animate-sidebar-neon-glow pointer-events-none"></div>

            {/* Decorative background elements */}
            <div className="absolute bottom-0 left-0 w-full h-48 overflow-hidden opacity-20 dark:opacity-15 pointer-events-none">
                <div className="absolute bottom-[-10px] left-[-10px] w-20 h-20 bg-white/30 rotate-45 transform animate-float"></div>
                <div className="absolute bottom-[-5px] left-[40px] w-12 h-12 bg-white/20 rotate-12 transform animate-float-reverse"></div>
                <div className="absolute bottom-[30px] left-[80px] w-16 h-16 bg-white/25 rotate-30 transform animate-float"></div>
                <div className="absolute bottom-[10px] left-[120px] w-24 h-24 bg-white/15 rotate-20 transform animate-float-reverse"></div>
                <div className="absolute bottom-[40px] left-[180px] w-14 h-14 bg-white/20 rotate-45 transform animate-float"></div>
                <div className="absolute bottom-[-20px] left-[220px] w-20 h-20 bg-white/10 rotate-30 transform animate-float-reverse"></div>
            </div>

            {/* Logo and title */}
            <div className={cn(
                "flex items-center gap-2 py-6 border-none transition-all duration-300",
                isCollapsed ? "flex-col justify-center px-0" : "flex-row px-6"
            )}>
                <div className="w-16 h-16 rounded-full flex items-center justify-center animate-pulse-glow">
                    <img src="/logo.svg" alt="logo" />
                </div>
                {!isCollapsed && (
                    <div className="font-bold text-xl">
                        <span className="text-[#8ED4FF] dark:text-[#A5DEFF] text-shadow-glow-blue transition-all duration-300">WAFHA</span>
                    </div>
                )}
            </div>

            {/* Toggle button */}
            <button
                onClick={toggleSidebar}
                className="absolute top-4 -right-3 w-6 h-6 bg-red-500 rounded-full flex items-center justify-center text-white z-50 border-2 border-white shadow-lg"
                style={{ backgroundColor: 'red' }} // Яркий цвет для видимости
            >
                {isCollapsed ? (
                    <ChevronRight className="w-4 h-4" />
                ) : (
                    <ChevronLeft className="w-4 h-4" />
                )}
            </button>

            {/* Navigation items */}
            <div className="flex-1 py-4">
                {sidebarItems
                    .filter((item) => item.display)
                    .map((item) => {
                        const isActive = currentFirstLevelPath === item.href
                        return (
                            <Link
                                key={item.href}
                                to={item.href}
                                className={cn(
                                    "flex items-center gap-3 font-medium py-3 w-full group transition-all duration-300 relative overflow-hidden",
                                    isActive
                                        ? "bg-white/15 dark:bg-white/10 shadow-[0_0_15px_rgba(255,255,255,0.4)] dark:shadow-[0_0_15px_rgba(255,255,255,0.25)] text-white"
                                        : "text-white/90 hover:text-white hover:shadow-[0_0_10px_rgba(255,255,255,0.3)] dark:hover:shadow-[0_0_12px_rgba(255,255,255,0.2)]",
                                    "before:absolute before:content-[''] before:top-0 before:left-0 before:w-full before:h-full before:bg-gradient-to-r before:from-white/5 before:to-white/20 dark:before:from-white/5 dark:before:to-white/15 before:transition-opacity before:duration-300",
                                    isActive
                                        ? "before:opacity-100"
                                        : "before:opacity-0 hover:before:opacity-100",
                                    isCollapsed ? "justify-center px-0" : "px-6"
                                )}
                                title={isCollapsed ? item.title : undefined}
                            >
                                <span className="relative z-10 flex items-center gap-3">
                                    <item.icon className={cn(
                                        "w-5 h-5 transition-transform",
                                        isActive ? "text-white" : "group-hover:animate-icon-shake"
                                    )} />
                                    {!isCollapsed && (
                                        <span className={cn(
                                            "transition-all dark:text-shadow-glow-white",
                                            isActive ? "font-semibold" : "group-hover:font-medium"
                                        )}>{item.title}</span>
                                    )}
                                </span>
                                <div className={cn(
                                    "absolute inset-0 bg-gradient-to-r from-transparent via-white/5 to-transparent blur-sm transition-opacity duration-500",
                                    isActive ? "opacity-70" : "opacity-0 group-hover:opacity-100"
                                )}></div>
                            </Link>
                        )
                    })}
            </div>

            {/* External Links */}
            <div className="py-4 relative z-10">
                <div className={cn(
                    "flex items-center transition-all duration-300",
                    isCollapsed ? "flex-col gap-2 px-0 justify-center" : "flex-row gap-4 px-6"
                )}>
                </div>
            </div>

            {/* Logout button */}
            <div className="mt-auto py-4 border-none">
                <button
                    onClick={handleLogout}
                    className={cn(
                        "flex items-center gap-3 font-medium py-3 w-full group transition-all duration-300 relative overflow-hidden",
                        isLogoutActive
                            ? "bg-white/15 dark:bg-white/10 shadow-[0_0_15px_rgba(255,255,255,0.4)] dark:shadow-[0_0_15px_rgba(255,255,255,0.25)] text-white"
                            : "text-white/90 hover:text-white hover:shadow-[0_0_10px_rgba(255,255,255,0.3)] dark:hover:shadow-[0_0_12px_rgba(255,255,255,0.2)]",
                        "before:absolute before:content-[''] before:top-0 before:left-0 before:w-full before:h-full before:bg-gradient-to-r before:from-white/5 before:to-white/20 dark:before:from-white/5 dark:before:to-white/15 before:transition-opacity before:duration-300",
                        isLogoutActive
                            ? "before:opacity-100"
                            : "before:opacity-0 hover:before:opacity-100",
                        isCollapsed ? "justify-center px-0" : "px-6"
                    )}
                    title={isCollapsed ? t("sidebar.logout") : undefined}
                >
                    <span className="relative z-10 flex items-center gap-3">
                        <LogOut className={cn(
                            "w-5 h-5 transition-transform",
                            isLogoutActive ? "text-white" : "group-hover:animate-icon-shake"
                        )} />
                        {!isCollapsed && (
                            <span className={cn(
                                "transition-all dark:text-shadow-glow-white",
                                isLogoutActive ? "font-semibold" : "group-hover:font-medium"
                            )}>{t("sidebar.logout")}</span>
                        )}
                    </span>
                    <div className={cn(
                        "absolute inset-0 bg-gradient-to-r from-transparent via-white/5 to-transparent blur-sm transition-opacity duration-500",
                        isLogoutActive ? "opacity-70" : "opacity-0 group-hover:opacity-100"
                    )}></div>
                </button>

                {!isCollapsed && (
                    <div className="text-center text-xs text-white/60 dark:text-white mt-4 px-4 flex items-center justify-center gap-1">
                        <a href="https://github.com/f1l88/wafha" target="_blank" rel="noopener noreferrer" className="text-white/60 dark:text-white dark:text-shadow-glow-white">WAFHA</a>
                    </div>
                )}
            </div>
        </div>
    )
}