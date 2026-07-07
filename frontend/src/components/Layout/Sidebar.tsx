import type {ComponentType} from 'react'
import {cn} from '@/lib/utils'
import {Wand2Icon, LayoutTemplateIcon, Plug2Icon, ScrollTextIcon} from 'lucide-react'
import logo from '@/assets/images/logo-universal.png'

export type ScreenKey = 'connect' | 'generate' | 'templates' | 'logs'

const SCREENS: { key: ScreenKey; label: string; icon: ComponentType<{ className?: string }> }[] = [
    {key: 'generate', label: 'Generate', icon: Wand2Icon},
    {key: 'templates', label: 'Templates', icon: LayoutTemplateIcon},
    {key: 'connect', label: 'Connect', icon: Plug2Icon},
    {key: 'logs', label: 'Logs', icon: ScrollTextIcon},
]

interface SidebarProps {
    active: ScreenKey
    onChange: (screen: ScreenKey) => void
}

export function Sidebar({active, onChange}: SidebarProps) {
    return (
        <div className="flex h-screen w-56 shrink-0 flex-col bg-sidebar text-sidebar-foreground">
            <div className="flex items-center gap-2 px-4 py-5">
                <img src={logo} alt="TicketSmith" className="size-7 rounded-lg shadow-sm" />
                <span className="font-heading text-sm font-semibold tracking-tight">TicketSmith</span>
            </div>

            <nav className="flex flex-1 flex-col gap-0.5 px-2.5">
                {SCREENS.map((s) => {
                    const isActive = active === s.key
                    const Icon = s.icon
                    return (
                        <button
                            key={s.key}
                            onClick={() => onChange(s.key)}
                            className={cn(
                                'group relative flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-sm font-medium transition-colors',
                                isActive
                                    ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                                    : 'text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground',
                            )}
                        >
                            <span
                                className={cn(
                                    'absolute inset-y-1 left-0 w-0.5 rounded-full bg-sidebar-primary transition-opacity',
                                    isActive ? 'opacity-100' : 'opacity-0',
                                )}
                            />
                            <Icon className={cn('size-4', isActive ? 'text-sidebar-primary' : 'opacity-80')} />
                            {s.label}
                        </button>
                    )
                })}
            </nav>

            <div className="px-4 py-3 text-xs text-sidebar-foreground/40">v1.0 · desktop</div>
        </div>
    )
}
