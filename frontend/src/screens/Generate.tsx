import {useEffect, useState} from 'react'
import {toast} from 'sonner'
import {connections, templates, tracker, ai} from '../../wailsjs/go/models'
import {BrowserOpenURL} from '../../wailsjs/runtime/runtime'
import {api} from '@/lib/api'
import {Card, CardContent, CardHeader, CardTitle, CardDescription} from '@/components/ui/card'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {Textarea} from '@/components/ui/textarea'
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select'
import {FormField} from '@/components/FormField'

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

export function Generate() {
    const [conns, setConns] = useState<Connection[]>([])
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

    const [rawInput, setRawInput] = useState('')
    const [generating, setGenerating] = useState(false)
    const [logId, setLogId] = useState<string | null>(null)
    const [ticket, setTicket] = useState<EditableTicket | null>(null)

    const [creating, setCreating] = useState(false)
    const [createdTicket, setCreatedTicket] = useState<tracker.Ticket | null>(null)

    const template = tmpls.find((t) => t.id === templateId) ?? null

    useEffect(() => {
        api.connections.list().then(setConns).catch((err) => toast.error(`Failed to load connections: ${err}`))
        api.templates.list().then(setTmpls).catch((err) => toast.error(`Failed to load templates: ${err}`))
    }, [])

    useEffect(() => {
        if (!connectionId) {
            setProjects([])
            setTypes([])
            return
        }
        api.tracker.projects(connectionId).then(setProjects).catch((err) => toast.error(`Failed to load projects: ${err}`))
        api.tracker.types(connectionId).then(setTypes).catch((err) => toast.error(`Failed to load ticket types: ${err}`))
        setProjectId('')
        setAssignees([])
        setAssigneeId(NONE)
    }, [connectionId])

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
            setTicket({
                subject: result.ticket.subject,
                description: result.ticket.description,
                fields: result.ticket.fields ?? {},
            })
        } catch (err) {
            toast.error(`Generation failed: ${err}`)
            setTicket(null)
            setLogId(null)
        } finally {
            setGenerating(false)
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
        <div className="grid gap-6 p-4">
            <Card>
                <CardHeader>
                    <CardTitle>Generate ticket</CardTitle>
                    <CardDescription>Paste rough notes and turn them into a structured, previewable ticket.</CardDescription>
                </CardHeader>
                <CardContent className="grid gap-4">
                    <div className="grid grid-cols-2 gap-4">
                        <FormField label="Connection">
                            <Select
                                value={connectionId}
                                onValueChange={(v) => setConnectionId(v as string)}
                                items={Object.fromEntries(conns.map((c) => [c.id, c.name]))}
                            >
                                <SelectTrigger><SelectValue placeholder="Select a connection" /></SelectTrigger>
                                <SelectContent>
                                    {conns.map((c) => <SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>)}
                                </SelectContent>
                            </Select>
                        </FormField>
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
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <FormField label="Project">
                            <Select
                                value={projectId}
                                onValueChange={(v) => setProjectId(v as string)}
                                disabled={!connectionId}
                                items={Object.fromEntries(projects.map((p) => [p.id, p.name]))}
                            >
                                <SelectTrigger><SelectValue placeholder="Select a project" /></SelectTrigger>
                                <SelectContent>
                                    {projects.map((p) => <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>)}
                                </SelectContent>
                            </Select>
                        </FormField>
                        <FormField label="Type">
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
                        <CardTitle>Preview</CardTitle>
                        <CardDescription>Review and edit before creating the ticket.</CardDescription>
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

                        <Button onClick={createTicket} disabled={!canCreate}>
                            {creating ? 'Creating…' : 'Create ticket'}
                        </Button>

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
    )
}
