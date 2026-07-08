import {useEffect, useState} from 'react'
import {RefreshCwIcon} from 'lucide-react'
import {toast} from 'sonner'
import {cn} from '@/lib/utils'
import {Button} from '@/components/ui/button'
import {Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter} from '@/components/ui/dialog'
import {ReleaseNotes} from '@/components/ReleaseNotes'
import {EventsOn} from '../../wailsjs/runtime/runtime'
import {CheckForUpdates, DownloadUpdate, InstallUpdate} from '../../wailsjs/go/main/App'
import type {updater} from '../../wailsjs/go/models'

type Phase = 'idle' | 'checking' | 'found' | 'downloading' | 'ready'

const LABEL: Record<Phase, string> = {
    idle: 'Check for updates',
    checking: 'Checking…',
    found: 'Check for updates',
    downloading: 'Downloading…',
    ready: 'Check for updates',
}

export function UpdateControl() {
    const [phase, setPhase] = useState<Phase>('idle')
    const [info, setInfo] = useState<updater.UpdateInfo | null>(null)
    const [progress, setProgress] = useState(0)
    const [pkgPath, setPkgPath] = useState('')
    const [installing, setInstalling] = useState(false)
    // Set when the silent startup check finds something. Startup never pops
    // a dialog on its own (that'd nag on every launch until you act on it) —
    // it just flips the button to say "Update available" with a dot. You
    // click it whenever you want; that always does a fresh check rather
    // than trusting this possibly-stale signal, so the dialog you actually
    // see is never out of date.
    const [pendingUpdate, setPendingUpdate] = useState(false)

    useEffect(() => {
        const offApplying = EventsOn('update:applying', () => {
            toast.info('Installing update — restarting in a moment...')
        })
        const offAvailable = EventsOn('update:available', () => {
            setPendingUpdate(true)
        })
        return () => {
            offApplying()
            offAvailable()
        }
    }, [])

    async function handleCheck() {
        setPendingUpdate(false)
        setPhase('checking')
        try {
            const result = await CheckForUpdates()
            if (!result) {
                toast.success("You're up to date")
                setPhase('idle')
                return
            }
            setInfo(result)
            setPhase('found')
        } catch {
            toast.error('Could not check for updates')
            setPhase('idle')
        }
    }

    async function handleUpdateNow() {
        if (!info) return
        setPhase('downloading')
        setProgress(0)
        const off = EventsOn('update:download-progress', (fraction: number) => setProgress(fraction))
        try {
            const path = await DownloadUpdate(info)
            setPkgPath(path)
            setPhase('ready')
        } catch {
            toast.error('Update download failed')
            setPhase('idle')
        } finally {
            off()
        }
    }

    async function handleInstall() {
        setInstalling(true)
        try {
            await InstallUpdate(pkgPath)
        } catch (err) {
            // Expected in `wails dev` — Velopack's Update.exe only exists next
            // to a real installed copy, not a dev build. Works in a packaged install.
            toast.error(`Install failed: ${err}`)
            setInstalling(false)
        }
    }

    const busy = phase === 'checking' || phase === 'downloading'
    const label = phase === 'downloading'
        ? `Downloading… ${Math.round(progress * 100)}%`
        : phase === 'idle' && pendingUpdate
            ? 'Update available'
            : LABEL[phase]

    return (
        <>
            <button
                onClick={handleCheck}
                disabled={busy}
                className={cn(
                    'flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-xs font-medium outline-none transition-colors hover:bg-sidebar-accent/60 focus-visible:text-sidebar-foreground disabled:pointer-events-none disabled:opacity-60',
                    phase === 'idle' && pendingUpdate
                        ? 'text-primary hover:text-primary'
                        : 'text-sidebar-foreground/60 hover:text-sidebar-foreground',
                )}
            >
                <span className="relative flex shrink-0">
                    <RefreshCwIcon className={cn('size-3.5', busy && 'animate-spin')} />
                    {phase === 'idle' && pendingUpdate && (
                        <span className="absolute -top-0.5 -right-0.5 size-1.5 rounded-full bg-primary ring-2 ring-sidebar" />
                    )}
                </span>
                {label}
            </button>

            <Dialog open={phase === 'found'} onOpenChange={(open) => !open && setPhase('idle')}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle className="flex items-center gap-2">
                            Update available
                            <span className="rounded-full bg-accent px-2 py-0.5 text-xs font-medium text-accent-foreground">
                                v{info?.Version}
                            </span>
                        </DialogTitle>
                    </DialogHeader>
                    {info?.ReleaseNotes ? (
                        <div className="max-h-72 overflow-y-auto overflow-x-hidden rounded-lg border bg-muted/30 p-3">
                            <ReleaseNotes markdown={info.ReleaseNotes} />
                        </div>
                    ) : (
                        <p className="text-sm text-muted-foreground">No release notes were provided for this version.</p>
                    )}
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setPhase('idle')}>Not now</Button>
                        <Button onClick={handleUpdateNow}>Update now</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={phase === 'ready'} onOpenChange={(open) => !open && setPhase('idle')}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Ready to install v{info?.Version}</DialogTitle>
                    </DialogHeader>
                    <p className="text-sm text-muted-foreground">
                        Restarting will close any unsaved work in progress.
                    </p>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setPhase('idle')} disabled={installing}>Later</Button>
                        <Button onClick={handleInstall} disabled={installing}>
                            {installing ? 'Restarting…' : 'Install & Restart'}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </>
    )
}
