import {useEffect, useState} from 'react'
import {toast} from 'sonner'
import {connections} from '../../wailsjs/go/models'
import {api} from '@/lib/api'
import {useConnections} from '@/lib/connections'
import {Card, CardContent, CardHeader, CardTitle, CardDescription} from '@/components/ui/card'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select'
import {Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetBody, SheetFooter} from '@/components/ui/sheet'
import {FormField} from '@/components/FormField'
import {DataTable, type DataTableColumn} from '@/components/DataTable'
import {ConfirmDialog} from '@/components/ConfirmDialog'
import {InfoTooltip} from '@/components/InfoTooltip'
import {PageHeader} from '@/components/Layout/PageHeader'
import {StatusBadge} from '@/components/StatusBadge'
import {Badge} from '@/components/ui/badge'
import {cn} from '@/lib/utils'
import {PlusIcon, FolderKanbanIcon, BotIcon, Plug2Icon, RefreshCwIcon, PlugZapIcon} from 'lucide-react'
import {JiraIcon, AzureDevOpsIcon} from '@/components/BrandIcons'
import type {ComponentType} from 'react'

type Connection = connections.Connection

const TRACKER_KINDS: {
    value: string
    label: string
    icon: ComponentType<{ className?: string }>
    tint: string
    enabled: boolean
}[] = [
    {value: 'openproject', label: 'OpenProject', icon: FolderKanbanIcon, tint: 'text-emerald-600 bg-emerald-500/12', enabled: true},
    {value: 'jira', label: 'Jira', icon: JiraIcon, tint: 'text-blue-700 bg-blue-600/10', enabled: false},
    {value: 'azuredevops', label: 'Azure DevOps', icon: AzureDevOpsIcon, tint: 'text-sky-700 bg-sky-600/10', enabled: false},
]

interface ConnectionFormState {
    name: string
    trackerKind: string
    baseUrl: string
    token: string
}

const emptyForm: ConnectionFormState = {name: '', trackerKind: 'openproject', baseUrl: '', token: ''}

function TrackerKindPicker({value, onChange, disabled}: { value: string; onChange: (v: string) => void; disabled?: boolean }) {
    return (
        <div className="grid grid-cols-3 gap-2">
            {TRACKER_KINDS.map((k) => {
                const Icon = k.icon
                const selected = value === k.value
                return (
                    <button
                        key={k.value}
                        type="button"
                        disabled={disabled || !k.enabled}
                        onClick={() => onChange(k.value)}
                        className={cn(
                            'relative flex flex-col items-center gap-1.5 rounded-lg border p-3 text-center transition-colors',
                            selected ? 'border-primary bg-accent ring-1 ring-primary' : 'border-border hover:bg-muted',
                            (disabled || !k.enabled) && 'cursor-not-allowed opacity-60 hover:bg-transparent',
                        )}
                    >
                        {!k.enabled && (
                            <Badge variant="outline" className="absolute -top-2 right-1 bg-background px-1.5 text-[10px]">
                                Coming soon
                            </Badge>
                        )}
                        <div className={cn('flex size-8 items-center justify-center rounded-md', k.tint)}>
                            <Icon className="size-4" />
                        </div>
                        <span className="text-xs font-medium">{k.label}</span>
                    </button>
                )
            })}
        </div>
    )
}

export function Connect() {
    const {connections: list, refresh} = useConnections()
    const [dialogOpen, setDialogOpen] = useState(false)
    const [editingId, setEditingId] = useState<string | null>(null)
    const [form, setForm] = useState<ConnectionFormState>(emptyForm)
    const [saving, setSaving] = useState(false)
    const [testingId, setTestingId] = useState<string | null>(null)
    const [testResults, setTestResults] = useState<Record<string, { ok: boolean; message: string }>>({})

    const [aiForm, setAiForm] = useState({baseUrl: '', model: '', apiKey: ''})
    const [aiHasKey, setAiHasKey] = useState(false)
    const [aiSaving, setAiSaving] = useState(false)
    const [aiModels, setAiModels] = useState<string[]>([])
    const [aiFetchingModels, setAiFetchingModels] = useState(false)
    const [aiTesting, setAiTesting] = useState(false)
    const [aiTestResult, setAiTestResult] = useState<{ ok: boolean; message: string } | null>(null)

    useEffect(() => {
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
            setTestResults((r) => ({...r, [conn.id]: {ok: true, message: 'Connected'}}))
        } catch (err) {
            toast.error(`"${conn.name}": ${err}`)
            setTestResults((r) => ({...r, [conn.id]: {ok: false, message: `${err}`}}))
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

    const fetchAiModels = async () => {
        setAiFetchingModels(true)
        try {
            const models = await api.aiSettings.listModels(aiForm.baseUrl, aiForm.apiKey)
            setAiModels(models)
            if (models.length === 0) {
                toast.error('That endpoint returned no models')
            } else {
                toast.success(`Found ${models.length} model${models.length === 1 ? '' : 's'}`)
                if (!models.includes(aiForm.model)) {
                    setAiForm((f) => ({...f, model: models[0]}))
                }
            }
        } catch (err) {
            toast.error(`Couldn't fetch models: ${err}`)
        } finally {
            setAiFetchingModels(false)
        }
    }

    const testAiConnection = async () => {
        setAiTesting(true)
        setAiTestResult(null)
        try {
            await api.aiSettings.test(aiForm.baseUrl, aiForm.model, aiForm.apiKey)
            setAiTestResult({ok: true, message: 'Connection works'})
            toast.success('AI connection test passed')
        } catch (err) {
            setAiTestResult({ok: false, message: `${err}`})
            toast.error(`AI connection test failed: ${err}`)
        } finally {
            setAiTesting(false)
        }
    }

    const columns: DataTableColumn<Connection>[] = [
        {key: 'name', header: 'Name', render: (c) => <span className="font-medium">{c.name}</span>},
        {
            key: 'kind',
            header: 'Tracker',
            render: (c) => {
                const kind = TRACKER_KINDS.find((k) => k.value === c.trackerKind)
                return (
                    <span className="inline-flex items-center gap-1.5">
                        {kind && <kind.icon className="size-3.5 text-muted-foreground" />}
                        {kind?.label ?? c.trackerKind}
                    </span>
                )
            },
        },
        {key: 'baseUrl', header: 'Base URL', render: (c) => <span className="text-muted-foreground">{c.baseUrl}</span>},
        {
            key: 'actions',
            header: '',
            className: 'text-right',
            render: (c) => {
                const result = testResults[c.id]
                return (
                    <div className="flex items-center justify-end gap-2">
                        {result && <StatusBadge status={result.ok ? 'success' : 'failure'} />}
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
                )
            },
        },
    ]

    return (
        <div className="flex flex-col">
            <PageHeader
                icon={Plug2Icon}
                title="Connections"
                description="Tracker instances and the AI provider TicketSmith uses to generate tickets."
            />
            <div className="grid gap-6 p-8">
                <Card>
                    <CardHeader>
                        <CardTitle>Tracker connections</CardTitle>
                        <CardDescription>Instances of your project tracker(s) that TicketSmith can file tickets into.</CardDescription>
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
                        <div className="flex items-center gap-2">
                            <div className="flex size-7 items-center justify-center rounded-md bg-accent text-accent-foreground">
                                <BotIcon className="size-4" />
                            </div>
                            <CardTitle>AI provider</CardTitle>
                        </div>
                        <CardDescription>
                            OpenAI-compatible endpoint used to generate tickets.
                            {aiHasKey && <span className="ml-1 font-medium text-foreground">An API key is currently saved.</span>}
                        </CardDescription>
                    </CardHeader>
                    <CardContent className="grid gap-4">
                        <div className="grid gap-4 sm:grid-cols-2">
                            <FormField label="Base URL" htmlFor="ai-base-url">
                                <Input
                                    id="ai-base-url"
                                    value={aiForm.baseUrl}
                                    onChange={(e) => setAiForm((f) => ({...f, baseUrl: e.target.value}))}
                                    placeholder="https://api.groq.com/openai/v1"
                                />
                            </FormField>
                            <FormField
                                label={
                                    <span className="inline-flex items-center gap-1.5">
                                        Model
                                        <InfoTooltip>
                                            Your API key doesn't determine which model runs. Each request has to name the exact model your provider expects.
                                        </InfoTooltip>
                                    </span>
                                }
                                htmlFor="ai-model"
                            >
                                {aiModels.length > 0 ? (
                                    <Select
                                        value={aiForm.model}
                                        onValueChange={(v) => setAiForm((f) => ({...f, model: v as string}))}
                                        items={Object.fromEntries(aiModels.map((m) => [m, m]))}
                                    >
                                        <SelectTrigger id="ai-model" className="w-full">
                                            <SelectValue placeholder="Select a model" />
                                        </SelectTrigger>
                                        <SelectContent>
                                            {aiModels.map((m) => <SelectItem key={m} value={m}>{m}</SelectItem>)}
                                        </SelectContent>
                                    </Select>
                                ) : (
                                    <Input
                                        id="ai-model"
                                        value={aiForm.model}
                                        onChange={(e) => setAiForm((f) => ({...f, model: e.target.value}))}
                                        placeholder="llama-3.1-8b-instant"
                                    />
                                )}
                            </FormField>
                        </div>
                        <FormField label="API key" htmlFor="ai-key" description={aiHasKey ? 'Leave blank to keep the saved key.' : undefined}>
                            <Input
                                id="ai-key"
                                type="password"
                                value={aiForm.apiKey}
                                onChange={(e) => setAiForm((f) => ({...f, apiKey: e.target.value}))}
                                placeholder={aiHasKey ? '••••••••' : 'sk-...'}
                            />
                        </FormField>
                        <div className="flex flex-wrap items-center gap-2">
                            <Button onClick={saveAiSettings} disabled={aiSaving}>
                                {aiSaving ? 'Saving…' : 'Save AI settings'}
                            </Button>
                            <Button
                                variant="outline"
                                disabled={aiFetchingModels || !aiForm.baseUrl}
                                onClick={fetchAiModels}
                            >
                                <RefreshCwIcon className={aiFetchingModels ? 'animate-spin' : ''} />
                                {aiFetchingModels ? 'Fetching…' : 'Fetch models'}
                            </Button>
                            <Button
                                variant="outline"
                                disabled={aiTesting || !aiForm.baseUrl || !aiForm.model}
                                onClick={testAiConnection}
                            >
                                <PlugZapIcon />
                                {aiTesting ? 'Testing…' : 'Test connection'}
                            </Button>
                            {aiTestResult && (
                                <span className="inline-flex items-center gap-1.5">
                                    <StatusBadge status={aiTestResult.ok ? 'success' : 'failure'} />
                                    {!aiTestResult.ok && <span className="text-xs text-destructive">{aiTestResult.message}</span>}
                                </span>
                            )}
                        </div>
                    </CardContent>
                </Card>
            </div>

            <Sheet open={dialogOpen} onOpenChange={setDialogOpen}>
                <SheetContent>
                    <SheetHeader>
                        <SheetTitle>{editingId ? 'Edit connection' : 'Add connection'}</SheetTitle>
                        <SheetDescription>
                            {editingId
                                ? "Update this tracker connection's details."
                                : 'Point TicketSmith at a tracker instance it can file tickets into.'}
                        </SheetDescription>
                    </SheetHeader>
                    <SheetBody>
                        <div className="grid gap-4">
                            <FormField label="Name" htmlFor="conn-name">
                                <Input id="conn-name" value={form.name} onChange={(e) => setForm((f) => ({...f, name: e.target.value}))} />
                            </FormField>
                            <FormField label="Tracker">
                                <TrackerKindPicker
                                    value={form.trackerKind}
                                    onChange={(v) => setForm((f) => ({...f, trackerKind: v}))}
                                    disabled={!!editingId}
                                />
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
                    </SheetBody>
                    <SheetFooter>
                        <Button onClick={save} disabled={saving}>{saving ? 'Saving…' : 'Save'}</Button>
                    </SheetFooter>
                </SheetContent>
            </Sheet>
        </div>
    )
}
