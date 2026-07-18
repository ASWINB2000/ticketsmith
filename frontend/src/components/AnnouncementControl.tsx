import {useEffect, useState} from 'react'
import {MegaphoneIcon} from 'lucide-react'
import {toast} from 'sonner'
import {cn} from '@/lib/utils'
import {api} from '@/lib/api'
import {Button} from '@/components/ui/button'
import {Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter} from '@/components/ui/dialog'
import {BrowserOpenURL, EventsOn} from '../../wailsjs/runtime/runtime'
import type {announcement} from '../../wailsjs/go/models'

export function AnnouncementControl() {
    // Non-null means "there's an announcement the user hasn't dismissed yet" —
    // drives both the sidebar affordance and the popup. Cleared only by
    // handleGotIt, never by closing the dialog, so closing it (backdrop/Esc)
    // just hides the popup for now; the sidebar button stays lit to reopen it.
    const [manifest, setManifest] = useState<announcement.Manifest | null>(null)
    const [dialogOpen, setDialogOpen] = useState(false)

    useEffect(() => {
        const offAvailable = EventsOn('announcement:available', (result: announcement.Manifest) => {
            setManifest(result)
            setDialogOpen(true)
        })
        // Also check on mount — covers the case where the silent startup
        // check's event fired before this listener mounted.
        api.announcement.check().then((result) => {
            if (result) {
                setManifest(result)
                setDialogOpen(true)
            }
        }).catch(() => {})
        return offAvailable
    }, [])

    async function handleGotIt() {
        if (!manifest) return
        const id = manifest.ID
        setDialogOpen(false)
        setManifest(null)
        try {
            await api.announcement.dismiss(id)
        } catch {
            toast.error("Couldn't save — this announcement may reappear next launch")
        }
    }

    if (!manifest) return null

    const warning = manifest.Level === 'warning'

    return (
        <>
            <button
                onClick={() => setDialogOpen(true)}
                className={cn(
                    'flex w-full items-center gap-1.5 rounded-md px-2 py-2 text-sm font-semibold outline-none transition-colors ring-1 ring-inset',
                    warning
                        ? 'bg-amber-400/10 text-amber-300 ring-amber-400/25 hover:bg-amber-400/15 hover:text-amber-200'
                        : 'bg-sky-400/10 text-sky-300 ring-sky-400/25 hover:bg-sky-400/15 hover:text-sky-200',
                )}
            >
                <span className="relative flex shrink-0">
                    <MegaphoneIcon className="size-4" />
                    <span className="absolute -top-0.5 -right-0.5 flex size-2">
                        <span className={cn('absolute inline-flex size-full animate-ping rounded-full opacity-75', warning ? 'bg-amber-400' : 'bg-sky-400')} />
                        <span className={cn('relative size-2 rounded-full ring-2 ring-sidebar', warning ? 'bg-amber-400' : 'bg-sky-400')} />
                    </span>
                </span>
                Announcement
            </button>

            <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle className="flex items-center gap-2">
                            <MegaphoneIcon className="size-4.5 shrink-0 text-sidebar-primary" />
                            {manifest.Title || 'Announcement'}
                        </DialogTitle>
                    </DialogHeader>
                    <p className="text-sm text-muted-foreground">{manifest.Body}</p>
                    <DialogFooter>
                        {manifest.URL && (
                            <Button variant="outline" onClick={() => BrowserOpenURL(manifest.URL)}>
                                Learn more
                            </Button>
                        )}
                        <Button onClick={handleGotIt}>Got it</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </>
    )
}
