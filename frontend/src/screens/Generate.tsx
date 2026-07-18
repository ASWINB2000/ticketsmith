import {useEffect, useRef, useState} from 'react'
import {toast} from 'sonner'
import {connections, templates, tracker, ai} from '../../wailsjs/go/models'
import {BrowserOpenURL} from '../../wailsjs/runtime/runtime'
import {api} from '@/lib/api'
import {useConnections} from '@/lib/connections'
import type {NotesPrefill} from '@/lib/notesPrefill'
import {Card, CardContent, CardHeader, CardTitle, CardDescription} from '@/components/ui/card'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {Textarea} from '@/components/ui/textarea'
import {NoteEditor} from '@/components/NoteEditor'
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select'
import {FormField} from '@/components/FormField'
import {PageHeader} from '@/components/Layout/PageHeader'
import {Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetBody, SheetFooter, SheetClose} from '@/components/ui/sheet'
import {Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter} from '@/components/ui/dialog'
import {Badge} from '@/components/ui/badge'
import {Wand2Icon, SettingsIcon, Undo2Icon, PaperclipIcon, XIcon, ImageIcon, VideoIcon, Trash2Icon, Loader2Icon} from 'lucide-react'

const IMAGE_EXTENSIONS = new Set(['.png', '.jpg', '.jpeg', '.gif', '.webp', '.heic', '.bmp'])

function isImagePath(path: string): boolean {
    return IMAGE_EXTENSIONS.has(path.slice(path.lastIndexOf('.')).toLowerCase())
}

type Connection = connections.Connection
type Template = templates.Template
type Project = tracker.Project
type TicketType = tracker.TicketType
type User = tracker.User
type Priority = tracker.Priority
type CustomFieldSchema = tracker.CustomFieldSchema

const NONE = '__none__'

interface EditableTicket {
    subject: string
    description: string
    fields: Record<string, string>
}

function sameTicket(a: EditableTicket, b: EditableTicket): boolean {
    return a.subject === b.subject && a.description === b.description
        && JSON.stringify(a.fields) === JSON.stringify(b.fields)
}

// Reads a pasted image/video File as base64 (stripping the data-URL prefix),
// since the Wails binding takes a plain string rather than raw bytes.
function fileToBase64(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
        const reader = new FileReader()
        reader.onload = () => resolve((reader.result as string).split(',')[1] ?? '')
        reader.onerror = () => reject(reader.error)
        reader.readAsDataURL(file)
    })
}

interface GenerateProps {
    active: boolean
    prefill: NotesPrefill | null
    onPrefillConsumed: () => void
}

export function Generate({active, prefill, onPrefillConsumed}: GenerateProps) {
    const {connections: conns} = useConnections()
    const [tmpls, setTmpls] = useState<Template[]>([])
    const [connectionId, setConnectionId] = useState('')
    const [templateId, setTemplateId] = useState('')

    const [projects, setProjects] = useState<Project[]>([])
    const [types, setTypes] = useState<TicketType[]>([])
    const [assignees, setAssignees] = useState<User[]>([])
    const [priorities, setPriorities] = useState<Priority[]>([])
    const [customFields, setCustomFields] = useState<CustomFieldSchema[]>([])
    const [projectId, setProjectId] = useState('')
    const [typeId, setTypeId] = useState('')
    const [assigneeId, setAssigneeId] = useState(NONE)
    const [priorityId, setPriorityId] = useState(NONE)
    const [parentId, setParentId] = useState('')
    const [startDate, setStartDate] = useState('')
    const [dueDate, setDueDate] = useState('')
    const [projectsError, setProjectsError] = useState<string | null>(null)
    const [typesError, setTypesError] = useState<string | null>(null)

    const [attachments, setAttachments] = useState<string[]>([])
    const [previews, setPreviews] = useState<Record<string, string>>({})
    const [uploadingAttachments, setUploadingAttachments] = useState(false)

    const [configOpen, setConfigOpen] = useState(false)
    // Last-used destination, restored once connections/projects finish loading.
    const savedDestination = useRef<{connectionId: string; projectId: string} | null>(null)

    const [rawInput, setRawInput] = useState('')
    const [generating, setGenerating] = useState(false)
    const [logId, setLogId] = useState<string | null>(null)
    const [ticket, setTicket] = useState<EditableTicket | null>(null)
    const [generatedTicket, setGeneratedTicket] = useState<EditableTicket | null>(null)
    // NoteEditor only reads its `content` prop on first mount, so a fresh
    // AI-regenerated draft (or a Reset to AI output) has to force a remount
    // via `key` to actually show the new markdown — bumped alongside ticket.
    const [previewRevision, setPreviewRevision] = useState(0)

    const [creating, setCreating] = useState(false)
    const [refining, setRefining] = useState(false)
    const [createdTicket, setCreatedTicket] = useState<tracker.Ticket | null>(null)

    // Notes -> Generate handoff (docs/NOTES_PLAN.md §5): which note(s) this
    // draft came from, if any, so a successful create can prompt keep/delete.
    const pendingNoteIds = useRef<string[]>([])
    const [notePromptOpen, setNotePromptOpen] = useState(false)
    const [notePromptIds, setNotePromptIds] = useState<string[]>([])

    const [clearConfirmOpen, setClearConfirmOpen] = useState(false)

    const template = tmpls.find((t) => t.id === templateId) ?? null
    const connectionName = conns.find((c) => c.id === connectionId)?.name
    const projectName = projects.find((p) => p.id === projectId)?.name
    const configured = !!connectionId && !!projectId
    const isEdited = !!ticket && !!generatedTicket && !sameTicket(ticket, generatedTicket)

    // Mirrors the case-insensitive name match CreateTicket performs
    // server-side (internal/tracker/openproject/workpackages.go), purely so
    // the Preview can show whether a field will land in a real tracker
    // custom field or fall back into the description.
    const matchedCustomField = (f: templates.Field) =>
        customFields.find((cf) => cf.name.toLowerCase() === (f.label || f.name).toLowerCase())

    // Fetches a thumbnail for a newly-staged image attachment; silently leaves
    // it out of `previews` on failure (e.g. too large) so the tile falls back
    // to a plain icon instead of erroring the whole attach flow.
    const loadPreview = (path: string) => {
        if (!isImagePath(path)) return
        api.generate.attachmentPreview(path)
            .then((dataUrl) => setPreviews((prev) => ({...prev, [path]: dataUrl})))
            .catch(() => {})
    }

    const stageAttachment = (path: string) => {
        setAttachments((prev) => (prev.includes(path) ? prev : [...prev, path]))
        loadPreview(path)
    }

    useEffect(() => {
        api.templates.list().then(setTmpls).catch((err) => toast.error(`Failed to load templates: ${err}`))
        api.generate.getDestination().then((d) => {
            if (d.connectionId && d.projectId) savedDestination.current = d
        }).catch(() => {})
    }, [])

    // Restore the saved connection once the connections list has loaded.
    useEffect(() => {
        const saved = savedDestination.current
        if (saved && conns.some((c) => c.id === saved.connectionId)) {
            setConnectionId(saved.connectionId)
        }
    }, [conns])

    // Generate stays mounted across tab switches (so notes/preview survive), but its
    // Sheet/Dialog render via a portal — hiding this screen alone wouldn't hide an open one.
    useEffect(() => {
        if (!active) {
            setConfigOpen(false)
            setNotePromptOpen(false)
            setClearConfirmOpen(false)
        }
    }, [active])

    // Lets pasting a screenshot (or a copied file from Explorer/Finder) anywhere
    // on this screen stage it as an attachment, without needing a dedicated
    // drop zone to focus first. Only active while this tab is the visible one,
    // and only intercepts the paste when it actually finds an image/video —
    // otherwise normal text paste into the Brief textarea is untouched.
    useEffect(() => {
        if (!active) return
        const handlePaste = (e: ClipboardEvent) => {
            const items = e.clipboardData?.items
            if (!items) return
            const fileItems = Array.from(items).filter(
                (it) => it.kind === 'file' && (it.type.startsWith('image/') || it.type.startsWith('video/')),
            )
            if (fileItems.length === 0) return
            e.preventDefault()
            fileItems.forEach(async (item, i) => {
                const file = item.getAsFile()
                if (!file) return
                try {
                    const base64 = await fileToBase64(file)
                    const ext = file.type.split('/')[1] ?? 'png'
                    const name = file.name || `pasted-${Date.now()}-${i}.${ext}`
                    const path = await api.generate.saveClipboardAttachment(base64, name)
                    stageAttachment(path)
                } catch (err) {
                    toast.error(`Failed to attach pasted file: ${err}`)
                }
            })
        }
        window.addEventListener('paste', handlePaste)
        return () => window.removeEventListener('paste', handlePaste)
    }, [active])

    // Consume a prefill handed off from the Notes screen: seed the raw input
    // and remember which note(s) it came from, starting a fresh conversion
    // (any stale preview from an unrelated prior session is cleared).
    useEffect(() => {
        if (!prefill) return
        setRawInput(prefill.content)
        pendingNoteIds.current = prefill.sourceNoteIds
        setTicket(null)
        setGeneratedTicket(null)
        setLogId(null)
        setCreatedTicket(null)
        setAttachments([])
        setPreviews({})
        onPrefillConsumed()
    }, [prefill])

    // Templates are otherwise only fetched once on mount, so refresh them whenever
    // this tab becomes active again to pick up edits made on the Templates screen.
    useEffect(() => {
        if (active) api.templates.list().then(setTmpls).catch((err) => toast.error(`Failed to load templates: ${err}`))
    }, [active])

    useEffect(() => {
        // Clear the previous connection's data immediately so stale projects/types
        // never linger while (or after) the new connection's fetch is in flight.
        setProjects([])
        setTypes([])
        setPriorities([])
        setProjectsError(null)
        setTypesError(null)
        setProjectId('')
        setAssignees([])
        setAssigneeId(NONE)
        setPriorityId(NONE)
        if (!connectionId) return

        api.tracker.projects(connectionId).then((ps) => {
            setProjects(ps)
            const saved = savedDestination.current
            if (saved && saved.connectionId === connectionId && ps.some((p) => p.id === saved.projectId)) {
                setProjectId(saved.projectId)
            }
        }).catch((err) => {
            setProjectsError(`${err}`)
            toast.error(`Failed to load projects: ${err}`)
        })
        api.tracker.types(connectionId).then(setTypes).catch((err) => {
            setTypesError(`${err}`)
            toast.error(`Failed to load ticket types: ${err}`)
        })
        api.tracker.priorities(connectionId).then(setPriorities).catch((err) => toast.error(`Failed to load priorities: ${err}`))
    }, [connectionId])

    // Remember the destination whenever both halves are set, so it survives a restart.
    useEffect(() => {
        if (connectionId && projectId) {
            api.generate.saveDestination(connectionId, projectId).catch(() => {})
        }
    }, [connectionId, projectId])

    useEffect(() => {
        if (!connectionId || !projectId) {
            setAssignees([])
            return
        }
        api.tracker.assignees(connectionId, projectId).then(setAssignees).catch((err) => toast.error(`Failed to load assignees: ${err}`))
        setAssigneeId(NONE)
    }, [connectionId, projectId])

    // Auto-select the tracker type matching the template's declared type name.
    useEffect(() => {
        if (!template || types.length === 0) return
        const match = types.find((t) => t.name.toLowerCase() === template.trackerTypeName.toLowerCase())
        if (match) setTypeId(match.id)
    }, [template, types])

    // Custom fields are scoped to a project+type combination — fetched purely
    // so the Preview can show which extraction fields will post to a real
    // tracker custom field vs. fall back into the description (see
    // internal/tracker/openproject's name-matching in CreateTicket).
    useEffect(() => {
        if (!connectionId || !projectId || !typeId) {
            setCustomFields([])
            return
        }
        api.tracker.customFields(connectionId, projectId, typeId).then((fields) => {
            console.log('[debug] tracker custom fields for', {projectId, typeId}, fields)
            setCustomFields(fields)
        }).catch((err) => {
            setCustomFields([])
            toast.error(`Failed to load custom fields: ${err}`)
        })
    }, [connectionId, projectId, typeId])

    const generate = async () => {
        if (!connectionId || !templateId || !rawInput.trim()) return
        setGenerating(true)
        setCreatedTicket(null)
        try {
            const result = await api.generate.run(connectionId, templateId, rawInput)
            setLogId(result.logId)
            const next: EditableTicket = {
                subject: result.ticket.subject,
                description: result.ticket.description,
                fields: result.ticket.fields ?? {},
            }
            setTicket(next)
            setGeneratedTicket(next)
            setPreviewRevision((r) => r + 1)
        } catch (err) {
            toast.error(`Generation failed: ${err}`)
            setTicket(null)
            setGeneratedTicket(null)
            setLogId(null)
        } finally {
            setGenerating(false)
        }
    }

    const resetToGenerated = () => {
        if (!generatedTicket) return
        setTicket(generatedTicket)
        setPreviewRevision((r) => r + 1)
    }

    // Unlike generate(), this elaborates on the *current* (possibly
    // hand-edited) draft rather than re-deriving one from rawInput alone —
    // so a point added directly in the Preview fields survives and gets
    // built out instead of being discarded on the next pass. On failure the
    // current draft is left untouched, since the whole point is not to lose
    // manual edits.
    const refine = async () => {
        if (!connectionId || !templateId || !rawInput.trim() || !ticket) return
        setRefining(true)
        setCreatedTicket(null)
        try {
            const current = new ai.StructuredTicket({subject: ticket.subject, description: ticket.description, fields: ticket.fields})
            const result = await api.generate.refine(connectionId, templateId, rawInput, current)
            setLogId(result.logId)
            const next: EditableTicket = {
                subject: result.ticket.subject,
                description: result.ticket.description,
                fields: result.ticket.fields ?? {},
            }
            setTicket(next)
            setGeneratedTicket(next)
            setPreviewRevision((r) => r + 1)
        } catch (err) {
            toast.error(`Refine failed: ${err}`)
        } finally {
            setRefining(false)
        }
    }

    const createTicket = async () => {
        if (!logId || !ticket || !connectionId || !projectId || !typeId) return
        setCreating(true)
        try {
            const payload = new ai.StructuredTicket({subject: ticket.subject, description: ticket.description, fields: ticket.fields})
            const result = await api.generate.create(
                logId, connectionId, projectId, typeId, payload,
                parentId.trim(), assigneeId === NONE ? '' : assigneeId,
                priorityId === NONE ? '' : priorityId, startDate, dueDate,
            )
            setCreatedTicket(result)
            toast.success('Ticket created')
            if (pendingNoteIds.current.length > 0) {
                setNotePromptIds(pendingNoteIds.current)
                setNotePromptOpen(true)
                pendingNoteIds.current = []
            }
            if (attachments.length > 0) {
                setUploadingAttachments(true)
                api.generate.uploadAttachments(connectionId, result.id, attachments)
                    .then(() => {
                        setAttachments([])
                        setPreviews({})
                    })
                    .catch((err) => toast.error(`Ticket created, but attaching files failed: ${err}`))
                    .finally(() => setUploadingAttachments(false))
            }
        } catch (err) {
            toast.error(`Create failed: ${err}`)
        } finally {
            setCreating(false)
        }
    }

    const addAttachments = async () => {
        try {
            const picked = await api.generate.pickAttachments()
            picked.forEach(stageAttachment)
        } catch (err) {
            toast.error(`Failed to open file picker: ${err}`)
        }
    }

    const removeAttachment = (path: string) => {
        setAttachments((prev) => prev.filter((p) => p !== path))
        setPreviews((prev) => {
            const {[path]: _removed, ...rest} = prev
            return rest
        })
        api.generate.discardClipboardAttachment(path).catch(() => {})
    }

    const keepNotes = () => {
        setNotePromptOpen(false)
        setNotePromptIds([])
    }

    const deleteNotes = async () => {
        try {
            await Promise.all(notePromptIds.map((id) => api.notes.remove(id)))
            toast.success(notePromptIds.length > 1 ? 'Notes deleted' : 'Note deleted')
        } catch (err) {
            toast.error(`Failed to delete note(s): ${err}`)
        } finally {
            setNotePromptOpen(false)
            setNotePromptIds([])
        }
    }

    const canGenerate = !!connectionId && !!templateId && rawInput.trim() !== '' && !generating && !refining
    const canCreate = !!ticket && !!projectId && !!typeId && !creating
    const canRefine = canGenerate && !!ticket
    const hasContent = rawInput.trim() !== '' || attachments.length > 0 || !!ticket || !!createdTicket

    // Resets everything about the current draft — brief, attachments, generated/edited
    // ticket, and per-ticket fields — but leaves the configured destination (connection/
    // project) alone since that's a standing preference, not part of this draft.
    const clearDraft = () => {
        attachments.forEach((path) => api.generate.discardClipboardAttachment(path).catch(() => {}))
        setRawInput('')
        setAttachments([])
        setPreviews({})
        setTicket(null)
        setGeneratedTicket(null)
        setLogId(null)
        setCreatedTicket(null)
        setTemplateId('')
        setTypeId('')
        setAssigneeId(NONE)
        setPriorityId(NONE)
        setParentId('')
        setStartDate('')
        setDueDate('')
        pendingNoteIds.current = []
        setClearConfirmOpen(false)
    }

    return (
        <div className="flex flex-col">
            <PageHeader
                icon={Wand2Icon}
                title="Generate"
                description="Paste rough notes and turn them into a structured, previewable ticket."
                actions={
                    <>
                        <Button variant="destructive" disabled={!hasContent} onClick={() => setClearConfirmOpen(true)}>
                            <Trash2Icon /> Clear
                        </Button>
                        <Button variant="outline" onClick={() => setConfigOpen(true)}>
                            <SettingsIcon />
                            {configured ? `${connectionName} · ${projectName}` : 'Configure destination'}
                        </Button>
                    </>
                }
            />
            <div className="grid gap-6 p-8">
            {!configured && (
                <div className="flex items-center justify-between gap-4 rounded-lg border border-dashed p-3">
                    <div>
                        <p className="text-sm font-medium">Set a connection and project to get started</p>
                        <p className="text-xs text-muted-foreground">TicketSmith needs to know where this ticket will be filed.</p>
                    </div>
                    <Button size="sm" onClick={() => setConfigOpen(true)}>
                        <SettingsIcon /> Configure
                    </Button>
                </div>
            )}
            <Card>
                <CardHeader>
                    <CardTitle>Brief</CardTitle>
                    <CardDescription>
                        Pick a template, then describe the issue or task in your own words.
                    </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-4">
                    <div className="grid grid-cols-2 gap-4">
                        <FormField label="Template" required>
                            <Select
                                value={templateId}
                                onValueChange={(v) => setTemplateId(v as string)}
                                items={Object.fromEntries(tmpls.map((t) => [t.id, t.name]))}
                            >
                                <SelectTrigger className="w-full"><SelectValue placeholder="Select a template" /></SelectTrigger>
                                <SelectContent>
                                    {tmpls.map((t) => <SelectItem key={t.id} value={t.id}>{t.name}</SelectItem>)}
                                </SelectContent>
                            </Select>
                        </FormField>
                        <FormField label="Type" required error={typesError ?? undefined}>
                            <Select
                                value={typeId}
                                onValueChange={(v) => setTypeId(v as string)}
                                disabled={!connectionId}
                                items={Object.fromEntries(types.map((t) => [t.id, t.name]))}
                            >
                                <SelectTrigger className="w-full"><SelectValue placeholder="Select a type" /></SelectTrigger>
                                <SelectContent>
                                    {types.map((t) => <SelectItem key={t.id} value={t.id}>{t.name}</SelectItem>)}
                                </SelectContent>
                            </Select>
                        </FormField>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <FormField label="Assignee">
                            <Select
                                value={assigneeId}
                                onValueChange={(v) => setAssigneeId(v as string)}
                                disabled={!projectId}
                                items={{[NONE]: 'Unassigned', ...Object.fromEntries(assignees.map((u) => [u.id, u.name]))}}
                            >
                                <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
                                <SelectContent>
                                    <SelectItem value={NONE}>Unassigned</SelectItem>
                                    {assignees.map((u) => <SelectItem key={u.id} value={u.id}>{u.name}</SelectItem>)}
                                </SelectContent>
                            </Select>
                        </FormField>
                        <FormField label="Parent ticket ID" htmlFor="parent-id">
                            <Input id="parent-id" value={parentId} onChange={(e) => setParentId(e.target.value)} placeholder="e.g. 123" />
                        </FormField>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <FormField label="Priority">
                            <Select
                                value={priorityId}
                                onValueChange={(v) => setPriorityId(v as string)}
                                disabled={!connectionId}
                                items={{[NONE]: 'Tracker default', ...Object.fromEntries(priorities.map((p) => [p.id, p.name]))}}
                            >
                                <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
                                <SelectContent>
                                    <SelectItem value={NONE}>Tracker default</SelectItem>
                                    {priorities.map((p) => <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>)}
                                </SelectContent>
                            </Select>
                        </FormField>
                        <FormField label="Dates">
                            <div className="grid grid-cols-2 gap-2">
                                <div className="grid gap-1">
                                    <span className="text-xs text-muted-foreground">Start</span>
                                    <Input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
                                </div>
                                <div className="grid gap-1">
                                    <span className="text-xs text-muted-foreground">Due</span>
                                    <Input type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} />
                                </div>
                            </div>
                        </FormField>
                    </div>

                    <FormField label="Brief" htmlFor="raw-input" required>
                        <Textarea
                            id="raw-input"
                            rows={5}
                            value={rawInput}
                            onChange={(e) => {
                                setRawInput(e.target.value)
                                if (e.target.value.trim() === '') pendingNoteIds.current = []
                            }}
                            placeholder="Paste rough notes about the issue or task here…"
                        />
                    </FormField>

                    <FormField label="Attachments" description="Paste a screenshot (Ctrl/Cmd+V) anywhere on this screen, or add files manually.">
                        <div className="flex flex-wrap items-end gap-3">
                            {attachments.map((path) => {
                                const name = path.split(/[\\/]/).pop()
                                const preview = previews[path]
                                return (
                                    <div key={path} className="group relative">
                                        <div className="flex size-16 items-center justify-center overflow-hidden rounded-lg border bg-muted">
                                            {preview ? (
                                                <img src={preview} alt={name} className="size-full object-cover" />
                                            ) : isImagePath(path) ? (
                                                <ImageIcon className="size-5 text-muted-foreground" />
                                            ) : (
                                                <VideoIcon className="size-5 text-muted-foreground" />
                                            )}
                                        </div>
                                        <button
                                            type="button"
                                            onClick={() => removeAttachment(path)}
                                            className="absolute -right-1.5 -top-1.5 flex size-5 items-center justify-center rounded-full border bg-background opacity-0 shadow-sm transition-opacity group-hover:opacity-100"
                                            aria-label={`Remove ${name}`}
                                        >
                                            <XIcon className="size-3" />
                                        </button>
                                        <p className="mt-1 w-16 truncate text-center text-[11px] text-muted-foreground" title={name}>{name}</p>
                                    </div>
                                )
                            })}
                            <Button type="button" variant="outline" size="sm" onClick={addAttachments}>
                                <PaperclipIcon /> Add images or videos
                            </Button>
                        </div>
                    </FormField>

                    <div className="flex gap-2">
                        <Button onClick={generate} disabled={!canGenerate} loading={generating}>
                            {generating ? 'Generating…' : ticket ? 'Regenerate from brief' : 'Generate'}
                        </Button>
                    </div>
                </CardContent>
            </Card>

            {ticket && (
                <Card>
                    <CardHeader>
                        <div className="flex items-center gap-2">
                            <CardTitle>Preview</CardTitle>
                            {isEdited && <Badge variant="secondary">Edited</Badge>}
                        </div>
                        <CardDescription>Edit anything below, then create the ticket. Tweak the Brief above and "Regenerate from brief" to start over, or edit a field here and "Regenerate from output" to elaborate on it in place.</CardDescription>
                    </CardHeader>
                    <CardContent className="grid gap-4">
                        <FormField label="Subject" htmlFor="preview-subject" required>
                            <Input
                                id="preview-subject"
                                value={ticket.subject}
                                onChange={(e) => setTicket((t) => t && {...t, subject: e.target.value})}
                            />
                        </FormField>
                        <FormField label="Description" htmlFor="preview-description">
                            <NoteEditor
                                key={`description-${previewRevision}`}
                                content={ticket.description}
                                onChange={(markdown) => setTicket((t) => t && {...t, description: markdown})}
                                className="min-h-32"
                            />
                        </FormField>
                        {template?.fieldsSchema.map((f) => (
                            <FormField
                                key={f.name}
                                label={
                                    <span className="inline-flex items-center gap-1.5">
                                        {f.label || f.name}
                                        {matchedCustomField(f) && (
                                            <Badge variant="outline" className="border-transparent bg-emerald-500/12 font-normal text-emerald-600">
                                                Custom field match
                                            </Badge>
                                        )}
                                    </span>
                                }
                                htmlFor={`preview-field-${f.name}`}
                            >
                                {f.type === 'textarea' ? (
                                    <NoteEditor
                                        key={`field-${f.name}-${previewRevision}`}
                                        content={ticket.fields[f.name] ?? ''}
                                        onChange={(markdown) => setTicket((t) => t && {...t, fields: {...t.fields, [f.name]: markdown}})}
                                        className="min-h-20"
                                    />
                                ) : (
                                    <Input
                                        id={`preview-field-${f.name}`}
                                        value={ticket.fields[f.name] ?? ''}
                                        onChange={(e) => setTicket((t) => t && {...t, fields: {...t.fields, [f.name]: e.target.value}})}
                                    />
                                )}
                            </FormField>
                        ))}

                        <div className="flex flex-wrap gap-2">
                            <Button onClick={createTicket} disabled={!canCreate} loading={creating}>
                                {creating ? 'Creating…' : 'Create ticket'}
                            </Button>
                            <Button variant="outline" onClick={refine} disabled={!canRefine} loading={refining} title="Elaborate on this draft as-is, including any edits you've made, without starting over">
                                {refining ? 'Refining…' : 'Regenerate from output'}
                            </Button>
                            {isEdited && (
                                <Button variant="ghost" onClick={resetToGenerated}>
                                    <Undo2Icon /> Reset to AI output
                                </Button>
                            )}
                        </div>

                        {createdTicket && (
                            <div className="rounded-lg border bg-muted/50 p-3 text-sm">
                                Created:{' '}
                                <button
                                    className="text-primary underline underline-offset-4"
                                    onClick={() => BrowserOpenURL(createdTicket.url)}
                                >
                                    {createdTicket.url}
                                </button>
                                {uploadingAttachments && (
                                    <span className="ml-2 inline-flex items-center gap-1.5 text-muted-foreground">
                                        <Loader2Icon className="size-3.5 animate-spin" /> Attaching files…
                                    </span>
                                )}
                            </div>
                        )}
                    </CardContent>
                </Card>
            )}
            </div>

            <Sheet open={configOpen} onOpenChange={setConfigOpen}>
                <SheetContent>
                    <SheetHeader>
                        <SheetTitle>Configure destination</SheetTitle>
                        <SheetDescription>Choose which tracker connection and project this ticket will go to.</SheetDescription>
                    </SheetHeader>
                    <SheetBody>
                        <div className="grid gap-4">
                            <FormField label="Connection">
                                <Select
                                    value={connectionId}
                                    onValueChange={(v) => setConnectionId(v as string)}
                                    items={Object.fromEntries(conns.map((c) => [c.id, c.name]))}
                                >
                                    <SelectTrigger className="w-full"><SelectValue placeholder="Select a connection" /></SelectTrigger>
                                    <SelectContent>
                                        {conns.map((c) => <SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>)}
                                    </SelectContent>
                                </Select>
                            </FormField>
                            <FormField label="Project" error={projectsError ?? undefined}>
                                <Select
                                    value={projectId}
                                    onValueChange={(v) => setProjectId(v as string)}
                                    disabled={!connectionId}
                                    items={Object.fromEntries(projects.map((p) => [p.id, p.name]))}
                                >
                                    <SelectTrigger className="w-full"><SelectValue placeholder="Select a project" /></SelectTrigger>
                                    <SelectContent>
                                        {projects.map((p) => <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>)}
                                    </SelectContent>
                                </Select>
                            </FormField>
                        </div>
                    </SheetBody>
                    <SheetFooter>
                        <SheetClose render={<Button />}>Done</SheetClose>
                    </SheetFooter>
                </SheetContent>
            </Sheet>

            <Dialog open={notePromptOpen} onOpenChange={setNotePromptOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Keep the originating note{notePromptIds.length > 1 ? 's' : ''}?</DialogTitle>
                        <DialogDescription>
                            This ticket was created from {notePromptIds.length > 1 ? `${notePromptIds.length} notes` : 'a note'} in
                            your inbox. Keep it in case it spawns another ticket later, or delete it now to keep the list tidy.
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={keepNotes}>Keep</Button>
                        <Button variant="destructive" onClick={deleteNotes}>Delete</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={clearConfirmOpen} onOpenChange={setClearConfirmOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Clear content?</DialogTitle>
                        <DialogDescription>
                            This clears the Brief text, attachments, and the generated Preview below. Your configured
                            connection and project stay set. This can't be undone.
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setClearConfirmOpen(false)}>Cancel</Button>
                        <Button variant="destructive" onClick={clearDraft}>Clear</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    )
}
