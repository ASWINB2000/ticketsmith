import {useEffect, useState, type ComponentType} from 'react'
import {cn} from '@/lib/utils'
import {Wand2Icon, LayoutTemplateIcon, Plug2Icon, ScrollTextIcon, CircleHelpIcon, NotebookPenIcon} from 'lucide-react'
import logo from '@/assets/images/logo-universal.png'
import {UpdateControl} from '@/components/UpdateControl'
import {ReleaseNotesButton} from '@/components/ReleaseNotesButton'
import {Version} from '../../../wailsjs/go/main/App'

export type ScreenKey = 'connect' | 'generate' | 'notes' | 'templates' | 'logs' | 'help'

const SCREENS: { key: ScreenKey; label: string; icon: ComponentType<{ className?: string }> }[] = [
    {key: 'generate', label: 'Generate', icon: Wand2Icon},
    {key: 'notes', label: 'Notes', icon: NotebookPenIcon},
    {key: 'templates', label: 'Templates', icon: LayoutTemplateIcon},
    {key: 'connect', label: 'Connect', icon: Plug2Icon},
    {key: 'logs', label: 'Logs', icon: ScrollTextIcon},
    {key: 'help', label: 'Help', icon: CircleHelpIcon},
]

interface SidebarProps {
    active: ScreenKey
    onChange: (screen: ScreenKey) => void
}

export function Sidebar({active, onChange}: SidebarProps) {
    const [version, setVersion] = useState('')

    useEffect(() => {
        Version().then(setVersion)
    }, [])

    return (
        <div className="flex h-screen w-56 shrink-0 flex-col bg-gradient-to-b from-sidebar to-[oklch(0.15_0.02_259)] text-sidebar-foreground">
            <div className="flex items-center gap-2 border-b border-sidebar-border px-4 py-5">
                <img src={logo} alt="TicketSmith" className="size-7 rounded-lg shadow-sm" />
                <div className="flex flex-col">
                    <span className="font-heading text-sm font-semibold tracking-tight">TicketSmith</span>
                    <ReleaseNotesButton version={version} />
                </div>
            </div>

            <nav className="flex flex-1 flex-col gap-0.5 px-2.5 pt-4">
                <span className="px-2.5 pb-1.5 text-[10px] font-semibold tracking-wider text-sidebar-foreground/40 uppercase">
                    Workspace
                </span>
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
                                    ? 'bg-sidebar-accent text-sidebar-accent-foreground ring-1 ring-inset ring-sidebar-border'
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

            <div className="border-t border-sidebar-border px-4 py-3">
                <UpdateControl />
            </div>
        </div>
    )
}
