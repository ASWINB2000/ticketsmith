import {useEffect, useState} from 'react'
import {MegaphoneIcon, XIcon} from 'lucide-react'
import {cn} from '@/lib/utils'
import {api} from '@/lib/api'
import {BrowserOpenURL, EventsOn} from '../../wailsjs/runtime/runtime'
import type {announcement} from '../../wailsjs/go/models'

export function AnnouncementBanner() {
    const [info, setInfo] = useState<announcement.Manifest | null>(null)

    useEffect(() => {
        const offAvailable = EventsOn('announcement:available', (result: announcement.Manifest) => {
            setInfo(result)
        })
        // Also check on demand — covers the case where the silent startup
        // check's event fired before this listener mounted.
        api.announcement.check().then((result) => {
            if (result) setInfo(result)
        }).catch(() => {})
        return offAvailable
    }, [])

    if (!info) return null

    const warning = info.Level === 'warning'

    async function handleDismiss() {
        const id = info!.ID
        setInfo(null)
        try {
            await api.announcement.dismiss(id)
        } catch {
            // best-effort — worst case it resurfaces next launch
        }
    }

    return (
        <div
            className={cn(
                'flex shrink-0 items-center gap-3 border-b px-4 py-2 text-sm',
                warning
                    ? 'border-amber-400/25 bg-amber-400/10 text-amber-200'
                    : 'border-sidebar-border bg-sidebar-accent/40 text-sidebar-foreground',
            )}
        >
            <MegaphoneIcon className="size-4 shrink-0" />
            <p className="min-w-0 flex-1">
                {info.Title && <span className="font-semibold">{info.Title}: </span>}
                {info.Body}
                {info.URL && (
                    <button
                        onClick={() => BrowserOpenURL(info.URL)}
                        className="ml-1.5 underline underline-offset-2 hover:no-underline"
                    >
                        Learn more
                    </button>
                )}
            </p>
            <button
                onClick={handleDismiss}
                className="shrink-0 rounded p-1 opacity-70 transition-opacity hover:opacity-100"
                aria-label="Dismiss announcement"
            >
                <XIcon className="size-4" />
            </button>
        </div>
    )
}
