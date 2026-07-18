import {useEffect, useState} from 'react'
import {toast} from 'sonner'
import {cn, formatRelativeTime} from '@/lib/utils'
import {notes as notesModel} from '../../wailsjs/go/models'
import {api} from '@/lib/api'
import type {NotesPrefill} from '@/lib/notesPrefill'
import {Card, CardContent, CardHeader} from '@/components/ui/card'
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
import {PageHeader} from '@/components/Layout/PageHeader'
import {ConfirmDialog} from '@/components/ConfirmDialog'
import {NotebookPenIcon, GitMergeIcon, SendIcon, Trash2Icon, Loader2Icon} from 'lucide-react'

type Note = notesModel.Note

interface NotesProps {
    onConvertToGenerate: (prefill: NotesPrefill) => void
}

function NoteCard({
    note,
    selected,
    onToggleSelect,
    onSaved,
    onDeleted,
    onConvert,
}: {
    note: Note
    selected: boolean
    onToggleSelect: () => void
    onSaved: () => void
    onDeleted: () => void
    onConvert: () => void
}) {
    // Seeded once from the note prop, like NoteEditor's content — a live
    // parent refresh shouldn't clobber an in-progress edit in this input.
    const [title, setTitle] = useState(note.title)

    const save = async (nextTitle: string, nextContent: string) => {
        // Idle focus/blur shouldn't bump updated_at and reshuffle the board.
        if (nextTitle === note.title && nextContent === note.content) return
        try {
            await api.notes.update(note.id, nextTitle, nextContent)
            onSaved()
        } catch (err) {
            toast.error(`Failed to update note: ${err}`)
        }
    }

    const remove = async () => {
        try {
            await api.notes.remove(note.id)
            toast.success('Note deleted')
            onDeleted()
        } catch (err) {
            toast.error(`Failed to delete note: ${err}`)
        }
    }

    return (
        <Card className={cn(selected && 'outline-2 outline-primary')}>
            <CardHeader className="flex flex-row items-start justify-between gap-2">
                <div className="flex min-w-0 flex-1 items-center gap-2">
                    <input
                        type="checkbox"
                        className="size-4 shrink-0"
                        checked={selected}
                        onChange={onToggleSelect}
                    />
                    <input
                        value={title}
                        onChange={(e) => setTitle(e.target.value)}
                        onBlur={() => save(title, note.content)}
                        placeholder="Untitled note"
                        className="min-w-0 flex-1 rounded px-1 -mx-1 font-heading text-sm font-semibold text-card-foreground outline-none placeholder:font-normal placeholder:text-muted-foreground focus:bg-muted"
                    />
                </div>
                <ConfirmDialog
                    trigger={
                        <Button size="icon-sm" variant="destructive" className="shrink-0" title="Delete note">
                            <Trash2Icon />
                        </Button>
                    }
                    title="Delete this note?"
                    description="This cannot be undone."
                    confirmLabel="Delete"
                    destructive
                    onConfirm={remove}
                />
            </CardHeader>
            <CardContent>
                <NoteEditor
                    content={note.content}
                    onBlur={(markdown) => save(title, markdown)}
                    className="max-h-56 border-none px-1 py-0.5 -mx-1 -my-0.5"
                />
                <div className="mt-3 flex items-center justify-between gap-2">
                    <span
                        className="text-xs text-muted-foreground"
                        title={new Date(note.updatedAt).toLocaleString()}
                    >
                        {formatRelativeTime(note.updatedAt)}
                    </span>
                    <Button size="sm" variant="outline" onClick={onConvert}>
                        <SendIcon /> Convert
                    </Button>
                </div>
            </CardContent>
        </Card>
    )
}

export function Notes({onConvertToGenerate}: NotesProps) {
    const [list, setList] = useState<Note[]>([])
    // Global quick-capture shortcut label (e.g. "Ctrl+T"), or '' when the
    // backend couldn't register one — in that case no hint is shown.
    const [captureShortcut, setCaptureShortcut] = useState('')
    const [newTitle, setNewTitle] = useState('')
    const [newContent, setNewContent] = useState('')
    const [newNoteKey, setNewNoteKey] = useState(0)
    const [saving, setSaving] = useState(false)

    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())

    // Merge preview state — shown in a Dialog since merging is a short-lived
    // action, not persistent board state. mergeDraftVersion is bumped only
    // when a fresh AI draft lands (start/regenerate), so it forces the
    // editor to remount with the new text without clobbering an in-progress
    // manual edit on every keystroke.
    const [mergeSourceIds, setMergeSourceIds] = useState<string[] | null>(null)
    const [mergeTitle, setMergeTitle] = useState('')
    const [mergeDraft, setMergeDraft] = useState('')
    const [mergeDraftVersion, setMergeDraftVersion] = useState(0)
    const [merging, setMerging] = useState(false)
    const [confirmingMerge, setConfirmingMerge] = useState(false)

    const refresh = () => {
        api.notes.list().then(setList).catch((err) => toast.error(`Failed to load notes: ${err}`))
    }
    useEffect(refresh, [])
    useEffect(() => {
        api.notes.quickCaptureShortcut().then(setCaptureShortcut).catch(() => {})
    }, [])

    const save = async () => {
        if (!newContent.trim()) return
        setSaving(true)
        try {
            await api.notes.create(newTitle.trim(), newContent.trim())
            setNewTitle('')
            setNewContent('')
            setNewNoteKey((k) => k + 1)
            refresh()
        } catch (err) {
            toast.error(`Failed to save note: ${err}`)
        } finally {
            setSaving(false)
        }
    }

    const toggleSelected = (id: string) => {
        setSelectedIds((prev) => {
            const next = new Set(prev)
            if (next.has(id)) next.delete(id)
            else next.add(id)
            return next
        })
    }

    const deselect = (id: string) => {
        setSelectedIds((prev) => {
            const next = new Set(prev)
            next.delete(id)
            return next
        })
    }

    // Merge requires 2+ selected cards — the board's Merge button stays
    // disabled below 2, so this is only ever called with >= 2 ids.
    const startMerge = async () => {
        const ids = Array.from(selectedIds)
        if (ids.length < 2) return
        setMergeSourceIds(ids)
        setMergeTitle('')
        setMerging(true)
        try {
            const draft = await api.notes.merge(ids)
            setMergeDraft(draft)
            setMergeDraftVersion((v) => v + 1)
        } catch (err) {
            toast.error(`Merge failed: ${err}`)
            setMergeSourceIds(null)
        } finally {
            setMerging(false)
        }
    }

    const regenerateMerge = async () => {
        if (!mergeSourceIds) return
        setMerging(true)
        try {
            const draft = await api.notes.merge(mergeSourceIds)
            setMergeDraft(draft)
            setMergeDraftVersion((v) => v + 1)
        } catch (err) {
            toast.error(`Merge failed: ${err}`)
        } finally {
            setMerging(false)
        }
    }

    const confirmMerge = async () => {
        if (!mergeSourceIds) return
        setConfirmingMerge(true)
        try {
            await api.notes.confirmMerge(mergeSourceIds, mergeTitle.trim(), mergeDraft)
            toast.success('Notes merged')
            setMergeSourceIds(null)
            setMergeTitle('')
            setMergeDraft('')
            setSelectedIds(new Set())
            refresh()
        } catch (err) {
            toast.error(`Failed to confirm merge: ${err}`)
        } finally {
            setConfirmingMerge(false)
        }
    }

    const cancelMerge = () => {
        setMergeSourceIds(null)
        setMergeTitle('')
        setMergeDraft('')
    }

    const convertToTicket = (n: Note) => {
        onConvertToGenerate({content: n.content, sourceNoteIds: [n.id]})
    }

    return (
        <div className="flex flex-col">
            <PageHeader
                icon={NotebookPenIcon}
                title="Notes"
                description="Jot something down instantly, decide later whether and how it becomes a ticket."
                actions={
                    captureShortcut ? (
                        <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
                            <kbd className="rounded border bg-muted px-1.5 py-0.5 font-mono text-[0.7rem] font-medium text-foreground">
                                {captureShortcut}
                            </kbd>
                            quick-captures a note from anywhere
                        </span>
                    ) : undefined
                }
            />
            <div className="grid gap-6 p-8">
                <Card>
                    <CardHeader>
                        <input
                            key={newNoteKey}
                            value={newTitle}
                            onChange={(e) => setNewTitle(e.target.value)}
                            placeholder="New note"
                            className="rounded px-1 -mx-1 font-heading text-base font-medium outline-none placeholder:font-medium placeholder:text-muted-foreground focus:bg-muted"
                        />
                    </CardHeader>
                    <CardContent className="grid gap-3">
                        <NoteEditor
                            key={newNoteKey}
                            content=""
                            onChange={setNewContent}
                            placeholder="Jot something down…"
                            className="min-h-40 resize-y flex-none"
                        />
                        <div>
                            <Button onClick={save} disabled={saving || !newContent.trim()} loading={saving}>
                                {saving ? 'Saving…' : 'Save'}
                            </Button>
                        </div>
                    </CardContent>
                </Card>

                <div>
                    <div className="mb-4 flex items-center justify-between gap-4">
                        <div>
                            <h2 className="font-heading text-base font-semibold">Board</h2>
                            <p className="text-sm text-muted-foreground">
                                Click into a note to edit it directly. Select at least 2 to merge them into one
                                ticket draft.
                            </p>
                        </div>
                        <Button variant="outline" onClick={startMerge} disabled={selectedIds.size < 2 || merging} loading={merging}>
                            {!merging && <GitMergeIcon />}
                            {merging ? 'Merging…' : `Merge${selectedIds.size >= 2 ? ` (${selectedIds.size})` : ''}`}
                        </Button>
                    </div>

                    {list.length === 0 ? (
                        <p className="rounded-md border border-dashed p-4 text-center text-sm text-muted-foreground">
                            No notes yet.
                        </p>
                    ) : (
                        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                            {list.map((n) => (
                                <NoteCard
                                    key={n.id}
                                    note={n}
                                    selected={selectedIds.has(n.id)}
                                    onToggleSelect={() => toggleSelected(n.id)}
                                    onSaved={refresh}
                                    onDeleted={() => {
                                        deselect(n.id)
                                        refresh()
                                    }}
                                    onConvert={() => convertToTicket(n)}
                                />
                            ))}
                        </div>
                    )}
                </div>
            </div>

            <Dialog open={mergeSourceIds !== null} onOpenChange={(open) => !open && cancelMerge()}>
                <DialogContent className="flex max-h-[85vh] w-full flex-col overflow-hidden sm:max-w-lg">
                    <DialogHeader>
                        <DialogTitle>Merge preview</DialogTitle>
                        <DialogDescription>
                            Edit anything below, then confirm or tweak nothing and regenerate for a different pass.
                        </DialogDescription>
                    </DialogHeader>
                    <Input
                        value={mergeTitle}
                        onChange={(e) => setMergeTitle(e.target.value)}
                        placeholder="Title (optional)"
                        className="shrink-0"
                    />
                    <div className="relative min-h-0 flex-1">
                        <NoteEditor
                            key={mergeDraftVersion}
                            content={mergeDraft}
                            onChange={setMergeDraft}
                            editable={!(merging && !mergeDraft)}
                            wrapperClassName="h-full"
                            className="min-h-0"
                        />
                        {merging && !mergeDraft && (
                            <div className="absolute inset-0 flex flex-col items-center justify-center gap-2.5 rounded-lg border border-input bg-background/80 text-sm text-muted-foreground backdrop-blur-sm">
                                <Loader2Icon className="size-5 animate-spin text-muted-foreground/70" />
                                Combining notes…
                            </div>
                        )}
                    </div>
                    <DialogFooter>
                        <Button variant="ghost" onClick={cancelMerge}>Cancel</Button>
                        <Button variant="outline" onClick={regenerateMerge} disabled={merging} loading={merging}>
                            {merging ? 'Regenerating…' : 'Regenerate'}
                        </Button>
                        <Button onClick={confirmMerge} disabled={confirmingMerge || merging || !mergeDraft.trim()} loading={confirmingMerge}>
                            {confirmingMerge ? 'Confirming…' : 'Confirm merge'}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    )
}
