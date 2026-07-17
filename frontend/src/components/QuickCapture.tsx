import {useEffect, useState} from 'react'
import {toast} from 'sonner'
import {EventsOn} from '../../wailsjs/runtime/runtime'
import {api} from '@/lib/api'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {NoteEditor} from '@/components/NoteEditor'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'
import {ZapIcon} from 'lucide-react'

// QuickCapture is the dialog behind the global Ctrl+Alt+N shortcut: the
// backend raises the window and emits "quickcapture:open" (see
// quickcapture_windows.go), and this pops a minimal title+content form that
// saves straight to the Notes board. It's mounted once in App.tsx, over
// whatever screen is active, so capture never disturbs in-progress work.
export function QuickCapture({onSaved}: {onSaved?: () => void}) {
    const [open, setOpen] = useState(false)
    // Bumped on every open so the title input and Tiptap editor remount
    // empty — NoteEditor only reads `content` on first mount.
    const [captureKey, setCaptureKey] = useState(0)
    const [title, setTitle] = useState('')
    const [content, setContent] = useState('')
    const [saving, setSaving] = useState(false)

    useEffect(() => {
        const off = EventsOn('quickcapture:open', () => {
            setTitle('')
            setContent('')
            setCaptureKey((k) => k + 1)
            setOpen(true)
        })
        return off
    }, [])

    const save = async () => {
        if (!content.trim()) return
        setSaving(true)
        try {
            await api.notes.create(title.trim(), content.trim())
            toast.success('Note captured')
            setOpen(false)
            onSaved?.()
        } catch (err) {
            toast.error(`Failed to save note: ${err}`)
        } finally {
            setSaving(false)
        }
    }

    return (
        <Dialog open={open} onOpenChange={setOpen}>
            <DialogContent className="flex max-h-[85vh] w-full flex-col sm:max-w-lg">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <ZapIcon className="size-4" /> Quick capture
                    </DialogTitle>
                    <DialogDescription>
                        Saved to your Notes board — convert it into a ticket whenever you're ready.
                    </DialogDescription>
                </DialogHeader>
                <Input
                    key={`title-${captureKey}`}
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    placeholder="Title (optional)"
                    className="shrink-0"
                    autoFocus
                />
                <NoteEditor
                    key={`content-${captureKey}`}
                    content=""
                    onChange={setContent}
                    placeholder="Jot something down…"
                    wrapperClassName="flex-1 min-h-0"
                    className="min-h-32"
                />
                <DialogFooter>
                    <Button variant="ghost" onClick={() => setOpen(false)}>Discard</Button>
                    <Button onClick={save} disabled={saving || !content.trim()}>
                        {saving ? 'Saving…' : 'Save note'}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    )
}
