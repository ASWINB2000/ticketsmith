import {Tabs, TabsList, TabsTrigger} from '@/components/ui/tabs'

export type ScreenKey = 'connect' | 'generate' | 'templates' | 'logs'

const SCREENS: { key: ScreenKey; label: string }[] = [
    {key: 'generate', label: 'Generate'},
    {key: 'templates', label: 'Templates'},
    {key: 'connect', label: 'Connect'},
    {key: 'logs', label: 'Logs'},
]

interface NavBarProps {
    active: ScreenKey
    onChange: (screen: ScreenKey) => void
}

export function NavBar({active, onChange}: NavBarProps) {
    return (
        <div className="flex items-center justify-between border-b px-4 py-2">
            <span className="font-heading text-sm font-semibold">Ticketsmith</span>
            <Tabs value={active} onValueChange={(v) => onChange(v as ScreenKey)}>
                <TabsList>
                    {SCREENS.map((s) => (
                        <TabsTrigger key={s.key} value={s.key}>{s.label}</TabsTrigger>
                    ))}
                </TabsList>
            </Tabs>
        </div>
    )
}
