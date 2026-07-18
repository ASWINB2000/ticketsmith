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
    // True whenever a known update hasn't been installed yet. Set by the
    // silent startup check (which also pops the dialog once) or by a manual
    // check finding something; only cleared when a fresh check comes back
    // empty or the update is actually installed. Dismissing the dialog with
    // "Not now"/"Later" must NOT clear this — the sidebar button should keep
    // saying "Update available" until you act on it, not just until you look
    // at it once.
    const [pendingUpdate, setPendingUpdate] = useState(false)

    useEffect(() => {
        const offApplying = EventsOn('update:applying', () => {
            toast.info('Installing update — restarting in a moment...')
        })
        const offAvailable = EventsOn('update:available', (result: updater.UpdateInfo) => {
            setInfo(result)
            setPendingUpdate(true)
            setPhase('found')
        })
        return () => {
            offApplying()
            offAvailable()
        }
    }, [])

    async function handleCheck() {
        setPhase('checking')
        try {
            const result = await CheckForUpdates()
            if (!result) {
                toast.success("You're up to date")
                setPendingUpdate(false)
                setPhase('idle')
                return
            }
            setInfo(result)
            setPendingUpdate(true)
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
    const attention = phase === 'idle' && pendingUpdate
    const label = phase === 'downloading'
        ? `Downloading… ${Math.round(progress * 100)}%`
        : attention
            ? 'Update available'
            : LABEL[phase]

    return (
        <>
            <button
                onClick={handleCheck}
                disabled={busy}
                className={cn(
                    'flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-xs font-medium outline-none transition-colors disabled:pointer-events-none disabled:opacity-60',
                    attention
                        ? 'bg-amber-400/10 py-2 text-sm font-semibold text-amber-300 ring-1 ring-inset ring-amber-400/25 hover:bg-amber-400/15 hover:text-amber-200 focus-visible:text-amber-200'
                        : 'text-sidebar-foreground/60 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground focus-visible:text-sidebar-foreground',
                )}
            >
                <span className="relative flex shrink-0">
                    <RefreshCwIcon className={cn(attention ? 'size-4' : 'size-3.5', busy && 'animate-spin')} />
                    {attention && (
                        <span className="absolute -top-0.5 -right-0.5 flex size-2">
                            <span className="absolute inline-flex size-full animate-ping rounded-full bg-amber-400 opacity-75" />
                            <span className="relative size-2 rounded-full bg-amber-400 ring-2 ring-sidebar" />
                        </span>
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
                        <Button onClick={handleInstall} disabled={installing} loading={installing}>
                            {installing ? 'Restarting…' : 'Install & Restart'}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </>
    )
}
