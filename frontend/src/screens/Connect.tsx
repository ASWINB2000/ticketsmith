import {useEffect, useState} from 'react'
import {toast} from 'sonner'
import {connections} from '../../wailsjs/go/models'
import {api} from '@/lib/api'
import {Card, CardContent, CardHeader, CardTitle, CardDescription} from '@/components/ui/card'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select'
import {Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog'
import {FormField} from '@/components/FormField'
import {DataTable, type DataTableColumn} from '@/components/DataTable'
import {ConfirmDialog} from '@/components/ConfirmDialog'
import {PlusIcon} from 'lucide-react'

type Connection = connections.Connection

const TRACKER_KINDS = [
    {value: 'openproject', label: 'OpenProject', enabled: true},
    {value: 'jira', label: 'Jira (coming soon)', enabled: false},
    {value: 'azuredevops', label: 'Azure DevOps (coming soon)', enabled: false},
]

interface ConnectionFormState {
    name: string
    trackerKind: string
    baseUrl: string
    token: string
}

const emptyForm: ConnectionFormState = {name: '', trackerKind: 'openproject', baseUrl: '', token: ''}

export function Connect() {
    const [list, setList] = useState<Connection[]>([])
    const [dialogOpen, setDialogOpen] = useState(false)
    const [editingId, setEditingId] = useState<string | null>(null)
    const [form, setForm] = useState<ConnectionFormState>(emptyForm)
    const [saving, setSaving] = useState(false)
    const [testingId, setTestingId] = useState<string | null>(null)

    const [aiForm, setAiForm] = useState({baseUrl: '', model: '', apiKey: ''})
    const [aiHasKey, setAiHasKey] = useState(false)
    const [aiSaving, setAiSaving] = useState(false)

    const refresh = () => {
        api.connections.list().then(setList).catch((err) => toast.error(`Failed to load connections: ${err}`))
    }

    useEffect(() => {
        refresh()
        api.aiSettings.get().then((s) => {
            setAiForm({baseUrl: s.baseUrl, model: s.model, apiKey: ''})
            setAiHasKey(s.hasKey)
        }).catch((err) => toast.error(`Failed to load AI settings: ${err}`))
    }, [])

    const openCreate = () => {
        setEditingId(null)
        setForm(emptyForm)
        setDialogOpen(true)
    }

    const openEdit = (conn: Connection) => {
        setEditingId(conn.id)
        setForm({name: conn.name, trackerKind: conn.trackerKind, baseUrl: conn.baseUrl, token: ''})
        setDialogOpen(true)
    }

    const save = async () => {
        setSaving(true)
        try {
            if (editingId) {
                await api.connections.update(editingId, form.name, form.baseUrl, form.token)
                toast.success('Connection updated')
            } else {
                await api.connections.create(form.name, form.trackerKind, form.baseUrl, form.token)
                toast.success('Connection created')
            }
            setDialogOpen(false)
            refresh()
        } catch (err) {
            toast.error(`${err}`)
        } finally {
            setSaving(false)
        }
    }

    const remove = async (id: string) => {
        try {
            await api.connections.remove(id)
            toast.success('Connection deleted')
            refresh()
        } catch (err) {
            toast.error(`${err}`)
        }
    }

    const test = async (conn: Connection) => {
        setTestingId(conn.id)
        try {
            await api.connections.test(conn.id)
            toast.success(`"${conn.name}": connection test passed`)
        } catch (err) {
            toast.error(`"${conn.name}": ${err}`)
        } finally {
            setTestingId(null)
        }
    }

    const saveAiSettings = async () => {
        setAiSaving(true)
        try {
            await api.aiSettings.save(aiForm.baseUrl, aiForm.model, aiForm.apiKey)
            toast.success('AI settings saved')
            setAiHasKey(aiHasKey || aiForm.apiKey !== '')
            setAiForm((f) => ({...f, apiKey: ''}))
        } catch (err) {
            toast.error(`${err}`)
        } finally {
            setAiSaving(false)
        }
    }

    const columns: DataTableColumn<Connection>[] = [
        {key: 'name', header: 'Name', render: (c) => c.name},
        {key: 'kind', header: 'Tracker', render: (c) => c.trackerKind},
        {key: 'baseUrl', header: 'Base URL', render: (c) => c.baseUrl},
        {
            key: 'actions',
            header: '',
            className: 'text-right',
            render: (c) => (
                <div className="flex justify-end gap-2">
                    <Button variant="outline" size="sm" disabled={testingId === c.id} onClick={() => test(c)}>
                        {testingId === c.id ? 'Testing…' : 'Test'}
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => openEdit(c)}>Edit</Button>
                    <ConfirmDialog
                        trigger={<Button variant="destructive" size="sm">Delete</Button>}
                        title={`Delete "${c.name}"?`}
                        description="This removes the connection and its stored credentials. This cannot be undone."
                        confirmLabel="Delete"
                        destructive
                        onConfirm={() => remove(c.id)}
                    />
                </div>
            ),
        },
    ]

    return (
        <div className="grid gap-6 p-4">
            <Card>
                <CardHeader>
                    <CardTitle>Tracker connections</CardTitle>
                    <CardDescription>Instances of your project tracker(s) that Ticketsmith can file tickets into.</CardDescription>
                </CardHeader>
                <CardContent className="grid gap-4">
                    <DataTable columns={columns} rows={list} rowKey={(c) => c.id} emptyMessage="No connections yet." />
                    <Button onClick={openCreate} className="justify-self-start">
                        <PlusIcon /> Add connection
                    </Button>
                </CardContent>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle>AI provider</CardTitle>
                    <CardDescription>
                        OpenAI-compatible endpoint used to generate tickets (e.g. Groq: https://api.groq.com/openai/v1).
                        {aiHasKey && <span className="ml-1 font-medium text-foreground">An API key is currently saved.</span>}
                    </CardDescription>
                </CardHeader>
                <CardContent className="grid max-w-md gap-4">
                    <FormField label="Base URL" htmlFor="ai-base-url">
                        <Input
                            id="ai-base-url"
                            value={aiForm.baseUrl}
                            onChange={(e) => setAiForm((f) => ({...f, baseUrl: e.target.value}))}
                            placeholder="https://api.groq.com/openai/v1"
                        />
                    </FormField>
                    <FormField label="Model" htmlFor="ai-model">
                        <Input
                            id="ai-model"
                            value={aiForm.model}
                            onChange={(e) => setAiForm((f) => ({...f, model: e.target.value}))}
                            placeholder="llama-3.1-8b-instant"
                        />
                    </FormField>
                    <FormField label="API key" htmlFor="ai-key" description={aiHasKey ? 'Leave blank to keep the saved key.' : undefined}>
                        <Input
                            id="ai-key"
                            type="password"
                            value={aiForm.apiKey}
                            onChange={(e) => setAiForm((f) => ({...f, apiKey: e.target.value}))}
                            placeholder={aiHasKey ? '••••••••' : 'sk-...'}
                        />
                    </FormField>
                    <Button onClick={saveAiSettings} disabled={aiSaving} className="justify-self-start">
                        {aiSaving ? 'Saving…' : 'Save AI settings'}
                    </Button>
                </CardContent>
            </Card>

            <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{editingId ? 'Edit connection' : 'Add connection'}</DialogTitle>
                    </DialogHeader>
                    <div className="grid gap-4">
                        <FormField label="Name" htmlFor="conn-name">
                            <Input id="conn-name" value={form.name} onChange={(e) => setForm((f) => ({...f, name: e.target.value}))} />
                        </FormField>
                        <FormField label="Tracker" htmlFor="conn-kind">
                            <Select
                                value={form.trackerKind}
                                onValueChange={(v) => setForm((f) => ({...f, trackerKind: v as string}))}
                                disabled={!!editingId}
                                items={Object.fromEntries(TRACKER_KINDS.map((k) => [k.value, k.label]))}
                            >
                                <SelectTrigger id="conn-kind"><SelectValue /></SelectTrigger>
                                <SelectContent>
                                    {TRACKER_KINDS.map((k) => (
                                        <SelectItem key={k.value} value={k.value} disabled={!k.enabled}>{k.label}</SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </FormField>
                        <FormField label="Base URL" htmlFor="conn-base-url">
                            <Input
                                id="conn-base-url"
                                value={form.baseUrl}
                                onChange={(e) => setForm((f) => ({...f, baseUrl: e.target.value}))}
                                placeholder="https://your-instance.example.com"
                            />
                        </FormField>
                        <FormField
                            label="API token"
                            htmlFor="conn-token"
                            description={editingId ? 'Leave blank to keep the current token.' : undefined}
                        >
                            <Input
                                id="conn-token"
                                type="password"
                                value={form.token}
                                onChange={(e) => setForm((f) => ({...f, token: e.target.value}))}
                            />
                        </FormField>
                    </div>
                    <DialogFooter>
                        <Button onClick={save} disabled={saving}>{saving ? 'Saving…' : 'Save'}</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    )
}
