import {useEffect, useRef, useState} from 'react'
import {toast} from 'sonner'
import {connections, templates, tracker, ai} from '../../wailsjs/go/models'
import {BrowserOpenURL} from '../../wailsjs/runtime/runtime'
import {api} from '@/lib/api'
import {useConnections} from '@/lib/connections'
import {Card, CardContent, CardHeader, CardTitle, CardDescription} from '@/components/ui/card'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {Textarea} from '@/components/ui/textarea'
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select'
import {FormField} from '@/components/FormField'
import {PageHeader} from '@/components/Layout/PageHeader'
import {Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetBody, SheetFooter, SheetClose} from '@/components/ui/sheet'
import {Badge} from '@/components/ui/badge'
import {Wand2Icon, SettingsIcon, Undo2Icon} from 'lucide-react'

type Connection = connections.Connection
type Template = templates.Template
type Project = tracker.Project
type TicketType = tracker.TicketType
type User = tracker.User

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

export function Generate({active}: { active: boolean }) {
    const {connections: conns} = useConnections()
    const [tmpls, setTmpls] = useState<Template[]>([])
    const [connectionId, setConnectionId] = useState('')
    const [templateId, setTemplateId] = useState('')

    const [projects, setProjects] = useState<Project[]>([])
    const [types, setTypes] = useState<TicketType[]>([])
    const [assignees, setAssignees] = useState<User[]>([])
    const [projectId, setProjectId] = useState('')
    const [typeId, setTypeId] = useState('')
    const [assigneeId, setAssigneeId] = useState(NONE)
    const [parentId, setParentId] = useState('')
    const [projectsError, setProjectsError] = useState<string | null>(null)
    const [typesError, setTypesError] = useState<string | null>(null)

    const [configOpen, setConfigOpen] = useState(false)
    // Last-used destination, restored once connections/projects finish loading.
    const savedDestination = useRef<{connectionId: string; projectId: string} | null>(null)

    const [rawInput, setRawInput] = useState('')
    const [generating, setGenerating] = useState(false)
    const [logId, setLogId] = useState<string | null>(null)
    const [ticket, setTicket] = useState<EditableTicket | null>(null)
    const [generatedTicket, setGeneratedTicket] = useState<EditableTicket | null>(null)

    const [creating, setCreating] = useState(false)
    const [createdTicket, setCreatedTicket] = useState<tracker.Ticket | null>(null)

    const template = tmpls.find((t) => t.id === templateId) ?? null
    const connectionName = conns.find((c) => c.id === connectionId)?.name
    const projectName = projects.find((p) => p.id === projectId)?.name
    const configured = !!connectionId && !!projectId
    const isEdited = !!ticket && !!generatedTicket && !sameTicket(ticket, generatedTicket)

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
    // Sheet renders via a portal — hiding this screen alone wouldn't hide an open one.
    useEffect(() => {
        if (!active) setConfigOpen(false)
    }, [active])

    useEffect(() => {
        // Clear the previous connection's data immediately so stale projects/types
        // never linger while (or after) the new connection's fetch is in flight.
        setProjects([])
        setTypes([])
        setProjectsError(null)
        setTypesError(null)
        setProjectId('')
        setAssignees([])
        setAssigneeId(NONE)
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
        if (generatedTicket) setTicket(generatedTicket)
    }

    const createTicket = async () => {
        if (!logId || !ticket || !connectionId || !projectId || !typeId) return
        setCreating(true)
        try {
            const payload = new ai.StructuredTicket({subject: ticket.subject, description: ticket.description, fields: ticket.fields})
            const result = await api.generate.create(
                logId, connectionId, projectId, typeId, payload,
                parentId.trim(), assigneeId === NONE ? '' : assigneeId,
            )
            setCreatedTicket(result)
            toast.success('Ticket created')
        } catch (err) {
            toast.error(`Create failed: ${err}`)
        } finally {
            setCreating(false)
        }
    }

    const canGenerate = !!connectionId && !!templateId && rawInput.trim() !== '' && !generating
    const canCreate = !!ticket && !!projectId && !!typeId && !creating

    return (
        <div className="flex flex-col">
            <PageHeader
                icon={Wand2Icon}
                title="Generate"
                description="Paste rough notes and turn them into a structured, previewable ticket."
                actions={
                    <Button variant="outline" onClick={() => setConfigOpen(true)}>
                        <SettingsIcon />
                        {configured ? `${connectionName} · ${projectName}` : 'Configure destination'}
                    </Button>
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
                    <CardTitle>Notes</CardTitle>
                    <CardDescription>Pick a template, then describe the issue or task in your own words.</CardDescription>
                </CardHeader>
                <CardContent className="grid gap-4">
                    <div className="grid grid-cols-2 gap-4">
                        <FormField label="Template">
                            <Select
                                value={templateId}
                                onValueChange={(v) => setTemplateId(v as string)}
                                items={Object.fromEntries(tmpls.map((t) => [t.id, t.name]))}
                            >
                                <SelectTrigger><SelectValue placeholder="Select a template" /></SelectTrigger>
                                <SelectContent>
                                    {tmpls.map((t) => <SelectItem key={t.id} value={t.id}>{t.name}</SelectItem>)}
                                </SelectContent>
                            </Select>
                        </FormField>
                        <FormField label="Type" error={typesError ?? undefined}>
                            <Select
                                value={typeId}
                                onValueChange={(v) => setTypeId(v as string)}
                                disabled={!connectionId}
                                items={Object.fromEntries(types.map((t) => [t.id, t.name]))}
                            >
                                <SelectTrigger><SelectValue placeholder="Select a type" /></SelectTrigger>
                                <SelectContent>
                                    {types.map((t) => <SelectItem key={t.id} value={t.id}>{t.name}</SelectItem>)}
                                </SelectContent>
                            </Select>
                        </FormField>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <FormField label="Assignee (optional)">
                            <Select
                                value={assigneeId}
                                onValueChange={(v) => setAssigneeId(v as string)}
                                disabled={!projectId}
                                items={{[NONE]: 'Unassigned', ...Object.fromEntries(assignees.map((u) => [u.id, u.name]))}}
                            >
                                <SelectTrigger><SelectValue /></SelectTrigger>
                                <SelectContent>
                                    <SelectItem value={NONE}>Unassigned</SelectItem>
                                    {assignees.map((u) => <SelectItem key={u.id} value={u.id}>{u.name}</SelectItem>)}
                                </SelectContent>
                            </Select>
                        </FormField>
                        <FormField label="Parent ticket ID (optional)" htmlFor="parent-id">
                            <Input id="parent-id" value={parentId} onChange={(e) => setParentId(e.target.value)} placeholder="e.g. 123" />
                        </FormField>
                    </div>

                    <FormField label="Notes" htmlFor="raw-input">
                        <Textarea
                            id="raw-input"
                            rows={5}
                            value={rawInput}
                            onChange={(e) => setRawInput(e.target.value)}
                            placeholder="Paste rough notes about the issue or task here…"
                        />
                    </FormField>

                    <div className="flex gap-2">
                        <Button onClick={generate} disabled={!canGenerate}>
                            {generating ? 'Generating…' : ticket ? 'Regenerate' : 'Generate'}
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
                        <CardDescription>Edit anything below, then create the ticket — or tweak your notes above and regenerate.</CardDescription>
                    </CardHeader>
                    <CardContent className="grid gap-4">
                        <FormField label="Subject" htmlFor="preview-subject">
                            <Input
                                id="preview-subject"
                                value={ticket.subject}
                                onChange={(e) => setTicket((t) => t && {...t, subject: e.target.value})}
                            />
                        </FormField>
                        <FormField label="Description" htmlFor="preview-description">
                            <Textarea
                                id="preview-description"
                                rows={6}
                                value={ticket.description}
                                onChange={(e) => setTicket((t) => t && {...t, description: e.target.value})}
                            />
                        </FormField>
                        {template?.fieldsSchema.map((f) => (
                            <FormField key={f.name} label={f.label || f.name} htmlFor={`preview-field-${f.name}`}>
                                {f.type === 'textarea' ? (
                                    <Textarea
                                        id={`preview-field-${f.name}`}
                                        rows={3}
                                        value={ticket.fields[f.name] ?? ''}
                                        onChange={(e) => setTicket((t) => t && {...t, fields: {...t.fields, [f.name]: e.target.value}})}
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
                            <Button onClick={createTicket} disabled={!canCreate}>
                                {creating ? 'Creating…' : 'Create ticket'}
                            </Button>
                            <Button variant="outline" onClick={generate} disabled={!canGenerate}>
                                {generating ? 'Regenerating…' : 'Regenerate'}
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
        </div>
    )
}
