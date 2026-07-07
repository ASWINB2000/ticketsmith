import {useEffect, useState} from 'react'
import {RefreshCwIcon} from 'lucide-react'
import {toast} from 'sonner'
import {cn} from '@/lib/utils'
import {Button} from '@/components/ui/button'
import {Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter} from '@/components/ui/dialog'
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

    useEffect(() => {
        return EventsOn('update:applying', () => {
            toast.info('Installing update — restarting in a moment...')
        })
    }, [])

    async function handleCheck() {
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
        await InstallUpdate(pkgPath)
    }

    const busy = phase === 'checking' || phase === 'downloading'
    const label = phase === 'downloading' ? `Downloading… ${Math.round(progress * 100)}%` : LABEL[phase]

    return (
        <>
            <button
                onClick={handleCheck}
                disabled={busy}
                className="flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-xs font-medium text-sidebar-foreground/60 outline-none transition-colors hover:bg-sidebar-accent/60 hover:text-sidebar-foreground focus-visible:text-sidebar-foreground disabled:pointer-events-none disabled:opacity-60"
            >
                <RefreshCwIcon className={cn('size-3.5', busy && 'animate-spin')} />
                {label}
            </button>

            <Dialog open={phase === 'found'} onOpenChange={(open) => !open && setPhase('idle')}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Update available — v{info?.Version}</DialogTitle>
                    </DialogHeader>
                    <p className="whitespace-pre-wrap text-sm text-muted-foreground">{info?.ReleaseNotes}</p>
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
                        <Button variant="outline" onClick={() => setPhase('idle')}>Later</Button>
                        <Button onClick={handleInstall}>Install &amp; Restart</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </>
    )
}
