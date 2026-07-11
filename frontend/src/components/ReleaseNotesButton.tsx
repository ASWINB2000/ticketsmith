import {useState} from 'react'
import {toast} from 'sonner'
import {SparklesIcon} from 'lucide-react'
import {api} from '@/lib/api'
import {cn, formatRelativeTime} from '@/lib/utils'
import type {updater} from '../../wailsjs/go/models'
import {BrowserOpenURL} from '../../wailsjs/runtime/runtime'
import {Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter} from '@/components/ui/dialog'
import {Button} from '@/components/ui/button'
import {ReleaseNotes} from '@/components/ReleaseNotes'
import {GitHubIcon} from '@/components/BrandIcons'

export function ReleaseNotesButton({version}: { version: string }) {
    const [open, setOpen] = useState(false)
    const [loading, setLoading] = useState(false)
    const [info, setInfo] = useState<updater.ReleaseNotesInfo | null>(null)
    // A zero-value Go time.Time (no published_at from the API) serializes to
    // "0001-01-01T00:00:00Z", which parses to a valid but pre-epoch Date —
    // getTime() > 0 filters that out rather than showing "56 years ago".
    const publishedDate = info?.PublishedAt && new Date(info.PublishedAt).getTime() > 0
        ? new Date(info.PublishedAt)
        : null

    const handleOpen = async () => {
        setOpen(true)
        if (info || loading) return
        setLoading(true)
        try {
            setInfo(await api.releaseNotes.latest())
        } catch (err) {
            toast.error(`Couldn't load release notes: ${err}`)
        } finally {
            setLoading(false)
        }
    }

    return (
        <>
            <button
                onClick={handleOpen}
                className={cn(
                    'group inline-flex w-fit items-center gap-1 rounded-full bg-sidebar-accent/50 px-1.5 py-0.5 ring-1 ring-inset ring-sidebar-border/60',
                    'text-[10px] font-medium tabular-nums text-sidebar-foreground/70 transition-colors',
                    'hover:bg-sidebar-accent hover:text-sidebar-foreground hover:ring-sidebar-primary/50',
                )}
            >
                <SparklesIcon className="size-2.5 shrink-0 text-sidebar-primary/80 transition-colors group-hover:text-sidebar-primary" />
                v{version}
            </button>

            <Dialog open={open} onOpenChange={setOpen}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle className="flex items-center gap-2">
                            Release notes
                            {info && (
                                <span className="rounded-full bg-accent px-2 py-0.5 text-xs font-medium text-accent-foreground">
                                    {info.Version}
                                </span>
                            )}
                        </DialogTitle>
                        {info?.AuthorLogin && (
                            <button
                                onClick={() => BrowserOpenURL(info.AuthorHTMLURL)}
                                className="flex w-fit items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground"
                            >
                                {info.AuthorAvatarURL && (
                                    <img src={info.AuthorAvatarURL} alt="" className="size-4 rounded-full" />
                                )}
                                Released by <span className="font-medium underline-offset-2 hover:underline">@{info.AuthorLogin}</span>
                                {publishedDate && (
                                    <span title={publishedDate.toLocaleString()}>
                                        · {formatRelativeTime(publishedDate)}
                                    </span>
                                )}
                            </button>
                        )}
                    </DialogHeader>
                    {loading ? (
                        <p className="text-sm text-muted-foreground">Loading…</p>
                    ) : info?.ReleaseNotes ? (
                        <div className="max-h-72 overflow-y-auto overflow-x-hidden rounded-lg border bg-muted/30 p-3">
                            <ReleaseNotes markdown={info.ReleaseNotes} />
                        </div>
                    ) : (
                        <p className="text-sm text-muted-foreground">No release notes available.</p>
                    )}
                    <DialogFooter>
                        {info?.HTMLURL && (
                            <Button variant="outline" onClick={() => BrowserOpenURL(info.HTMLURL)}>
                                <GitHubIcon className="size-4" /> View on GitHub
                            </Button>
                        )}
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </>
    )
}
