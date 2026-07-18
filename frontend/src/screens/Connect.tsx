import {useEffect, useState} from 'react'
import {toast} from 'sonner'
import {connections} from '../../wailsjs/go/models'
import {api} from '@/lib/api'
import {useConnections} from '@/lib/connections'
import {Card, CardContent, CardHeader, CardTitle, CardDescription, CardAction} from '@/components/ui/card'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select'
import {Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetBody, SheetFooter} from '@/components/ui/sheet'
import {Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription} from '@/components/ui/dialog'
import {FormField} from '@/components/FormField'
import {DataTable, type DataTableColumn} from '@/components/DataTable'
import {ConfirmDialog} from '@/components/ConfirmDialog'
import {InfoTooltip} from '@/components/InfoTooltip'
import {PageHeader} from '@/components/Layout/PageHeader'
import {StatusBadge} from '@/components/StatusBadge'
import {Badge} from '@/components/ui/badge'
import {cn} from '@/lib/utils'
import {PlusIcon, BotIcon, Plug2Icon, RefreshCwIcon, PlugZapIcon, GaugeIcon, CheckIcon} from 'lucide-react'
import {JiraIcon, AzureDevOpsIcon, OpenProjectIcon} from '@/components/BrandIcons'
import {LoadingPlaceholder} from '@/components/LoadingPlaceholder'
import {main, ai, aiusage} from '../../wailsjs/go/models'
import type {ComponentType} from 'react'

function UsageBar({label, remaining, limit, windowSeconds}: {label: string; remaining: number; limit: number; windowSeconds: number}) {
    const used = Math.max(limit - remaining, 0)
    const pct = limit > 0 ? Math.min((used / limit) * 100, 100) : 0
    return (
        <div className="grid gap-1.5">
            <div className="flex items-center justify-between text-sm">
                <span className="font-medium">
                    {label}
                    {windowSeconds > 0 && <span className="ml-1 font-normal text-muted-foreground">(per {windowSeconds}s window)</span>}
                </span>
                <span className="text-muted-foreground">{remaining.toLocaleString()} / {limit.toLocaleString()} remaining</span>
            </div>
            <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                <div className="h-full rounded-full bg-primary" style={{width: `${pct}%`}} />
            </div>
        </div>
    )
}

// Recorded-usage rollup: Ticketsmith's own count of what it has spent against
// the endpoint, from the "usage" object in every completion response.
function RecordedUsageTable({summary}: {summary: aiusage.Summary}) {
    const rows: {label: string; totals: aiusage.PeriodTotals}[] = [
        {label: 'Today', totals: summary.today},
        {label: 'Last 7 days', totals: summary.last7Days},
        {label: 'All time', totals: summary.allTime},
    ]
    return (
        <div className="grid gap-1 text-sm">
            <div className="grid grid-cols-3 gap-2 border-b pb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                <span />
                <span className="text-right">Requests</span>
                <span className="text-right">Tokens</span>
            </div>
            {rows.map(({label, totals}) => (
                <div key={label} className="grid grid-cols-3 gap-2 py-0.5">
                    <span className="font-medium">{label}</span>
                    <span className="text-right tabular-nums">{totals.requests.toLocaleString()}</span>
                    <span className="text-right tabular-nums">{totals.totalTokens.toLocaleString()}</span>
                </div>
            ))}
        </div>
    )
}

type Connection = connections.Connection
type AIProfile = ai.Profile

// Quick-fill presets for the AI profile sheet — the providers the Help
// screen documents, minus retired ones. Just fills the form; nothing is
// saved until the user does.
const AI_PRESETS: {label: string; name: string; baseUrl: string; model: string}[] = [
    {label: 'Groq', name: 'Groq', baseUrl: 'https://api.groq.com/openai/v1', model: 'openai/gpt-oss-120b'},
    {label: 'Gemini', name: 'Gemini', baseUrl: 'https://generativelanguage.googleapis.com/v1beta/openai', model: 'gemini-3.5-flash'},
    {label: 'OpenAI', name: 'OpenAI', baseUrl: 'https://api.openai.com/v1', model: 'gpt-4.1-mini'},
    {label: 'Ollama (local)', name: 'Local Ollama', baseUrl: 'http://localhost:11434/v1', model: ''},
]

interface AIProfileFormState {
    name: string
    baseUrl: string
    model: string
    apiKey: string
}

const emptyAiForm: AIProfileFormState = {name: '', baseUrl: '', model: '', apiKey: ''}

const TRACKER_KINDS: {
    value: string
    label: string
    icon: ComponentType<{ className?: string }>
    tint: string
    enabled: boolean
}[] = [
    {value: 'openproject', label: 'OpenProject', icon: OpenProjectIcon, tint: 'text-sky-700 bg-sky-600/10', enabled: true},
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

    const [aiProfiles, setAiProfiles] = useState<AIProfile[]>([])
    const [aiSheetOpen, setAiSheetOpen] = useState(false)
    const [editingAiId, setEditingAiId] = useState<string | null>(null)
    const [aiForm, setAiForm] = useState<AIProfileFormState>(emptyAiForm)
    const [aiHasKey, setAiHasKey] = useState(false)
    const [aiSaving, setAiSaving] = useState(false)
    const [aiModels, setAiModels] = useState<string[]>([])
    const [aiFetchingModels, setAiFetchingModels] = useState(false)
    const [aiTesting, setAiTesting] = useState(false)
    const [aiTestResult, setAiTestResult] = useState<{ ok: boolean; message: string } | null>(null)
    const [aiActivatingId, setAiActivatingId] = useState<string | null>(null)
    // Usage dialog is scoped to the profile whose Usage button was clicked.
    const [aiUsageProfile, setAiUsageProfile] = useState<AIProfile | null>(null)
    const [aiUsageLoading, setAiUsageLoading] = useState(false)
    const [aiUsage, setAiUsage] = useState<main.AIUsageView | null>(null)
    const [aiUsageError, setAiUsageError] = useState<string | null>(null)

    const refreshAiProfiles = () => {
        api.aiProfiles.list().then(setAiProfiles).catch((err) => toast.error(`Failed to load AI profiles: ${err}`))
    }
    useEffect(refreshAiProfiles, [])

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

    const openAiCreate = () => {
        setEditingAiId(null)
        setAiForm(emptyAiForm)
        setAiHasKey(false)
        setAiModels([])
        setAiTestResult(null)
        setAiSheetOpen(true)
    }

    const openAiEdit = (p: AIProfile) => {
        setEditingAiId(p.id)
        setAiForm({name: p.name, baseUrl: p.baseUrl, model: p.model, apiKey: ''})
        setAiHasKey(p.hasKey)
        setAiModels([])
        setAiTestResult(null)
        setAiSheetOpen(true)
    }

    const saveAiProfile = async () => {
        setAiSaving(true)
        try {
            if (editingAiId) {
                await api.aiProfiles.update(editingAiId, aiForm.name, aiForm.baseUrl, aiForm.model, aiForm.apiKey)
                toast.success('AI profile updated')
            } else {
                await api.aiProfiles.create(aiForm.name, aiForm.baseUrl, aiForm.model, aiForm.apiKey)
                toast.success('AI profile created')
            }
            setAiSheetOpen(false)
            refreshAiProfiles()
        } catch (err) {
            toast.error(`${err}`)
        } finally {
            setAiSaving(false)
        }
    }

    const removeAiProfile = async (p: AIProfile) => {
        try {
            await api.aiProfiles.remove(p.id)
            toast.success(`AI profile "${p.name}" deleted`)
            refreshAiProfiles()
        } catch (err) {
            toast.error(`${err}`)
        }
    }

    const activateAiProfile = async (p: AIProfile) => {
        setAiActivatingId(p.id)
        try {
            await api.aiProfiles.setActive(p.id)
            toast.success(`"${p.name}" is now the active AI provider`)
            refreshAiProfiles()
        } catch (err) {
            toast.error(`${err}`)
        } finally {
            setAiActivatingId(null)
        }
    }

    const fetchAiModels = async () => {
        setAiFetchingModels(true)
        try {
            const models = await api.aiProfiles.listModels(editingAiId ?? '', aiForm.baseUrl, aiForm.apiKey)
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
            await api.aiProfiles.test(editingAiId ?? '', aiForm.baseUrl, aiForm.model, aiForm.apiKey)
            setAiTestResult({ok: true, message: 'Connection works'})
            toast.success('AI connection test passed')
        } catch (err) {
            setAiTestResult({ok: false, message: `${err}`})
            toast.error(`AI connection test failed: ${err}`)
        } finally {
            setAiTesting(false)
        }
    }

    const checkAiUsage = async (p: AIProfile) => {
        setAiUsageProfile(p)
        setAiUsageLoading(true)
        setAiUsage(null)
        setAiUsageError(null)
        try {
            const usage = await api.aiProfiles.usage(p.id, p.baseUrl, p.model, '')
            setAiUsage(usage)
        } catch (err) {
            setAiUsageError(`${err}`)
        } finally {
            setAiUsageLoading(false)
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
                        <Button variant="outline" size="sm" disabled={testingId === c.id} loading={testingId === c.id} onClick={() => test(c)}>
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

    const aiColumns: DataTableColumn<AIProfile>[] = [
        {
            key: 'name',
            header: 'Name',
            render: (p) => (
                <span className="inline-flex items-center gap-2 font-medium">
                    {p.name}
                    {p.active && <Badge className="gap-1"><CheckIcon className="size-3" /> Active</Badge>}
                </span>
            ),
        },
        {key: 'model', header: 'Model', render: (p) => <span className="text-muted-foreground">{p.model || '—'}</span>},
        {
            key: 'actions',
            header: '',
            className: 'text-right',
            render: (p) => (
                <div className="flex items-center justify-end gap-2">
                    {!p.active && (
                        <Button
                            variant="outline"
                            size="sm"
                            disabled={aiActivatingId === p.id}
                            loading={aiActivatingId === p.id}
                            onClick={() => activateAiProfile(p)}
                        >
                            {aiActivatingId === p.id ? 'Switching…' : 'Use'}
                        </Button>
                    )}
                    <Button variant="outline" size="sm" title="Recorded spend and provider rate limits" onClick={() => checkAiUsage(p)}>
                        <GaugeIcon />
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => openAiEdit(p)}>Edit</Button>
                    <ConfirmDialog
                        trigger={<Button variant="destructive" size="sm">Delete</Button>}
                        title={`Delete "${p.name}"?`}
                        description="This removes the profile and its stored API key. This cannot be undone."
                        confirmLabel="Delete"
                        destructive
                        onConfirm={() => removeAiProfile(p)}
                    />
                </div>
            ),
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
                            <CardTitle>AI providers</CardTitle>
                        </div>
                        <CardDescription>
                            Saved OpenAI-compatible profiles. The active one is used for every generation. Switch with one click when a provider is slow, throttled, or retired.
                        </CardDescription>
                    </CardHeader>
                    <CardContent className="grid gap-4">
                        <DataTable columns={aiColumns} rows={aiProfiles} rowKey={(p) => p.id} emptyMessage="No AI profiles yet." />
                        <Button onClick={openAiCreate} className="justify-self-start">
                            <PlusIcon /> Add profile
                        </Button>
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
                        <Button onClick={save} disabled={saving} loading={saving}>{saving ? 'Saving…' : 'Save'}</Button>
                    </SheetFooter>
                </SheetContent>
            </Sheet>

            <Sheet open={aiSheetOpen} onOpenChange={setAiSheetOpen}>
                <SheetContent>
                    <SheetHeader>
                        <SheetTitle>{editingAiId ? 'Edit AI profile' : 'Add AI profile'}</SheetTitle>
                        <SheetDescription>
                            {editingAiId
                                ? "Update this provider profile's details."
                                : 'Save an OpenAI-compatible provider you can switch to anytime.'}
                        </SheetDescription>
                    </SheetHeader>
                    <SheetBody>
                        <div className="grid gap-4">
                            {!editingAiId && (
                                <FormField label="Quick fill">
                                    <div className="flex flex-wrap gap-2">
                                        {AI_PRESETS.map((preset) => (
                                            <Button
                                                key={preset.label}
                                                type="button"
                                                variant="outline"
                                                size="sm"
                                                onClick={() => setAiForm((f) => ({
                                                    ...f,
                                                    name: f.name || preset.name,
                                                    baseUrl: preset.baseUrl,
                                                    model: preset.model,
                                                }))}
                                            >
                                                {preset.label}
                                            </Button>
                                        ))}
                                    </div>
                                </FormField>
                            )}
                            <FormField label="Name" htmlFor="ai-name">
                                <Input
                                    id="ai-name"
                                    value={aiForm.name}
                                    onChange={(e) => setAiForm((f) => ({...f, name: e.target.value}))}
                                    placeholder="Groq"
                                />
                            </FormField>
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
                                        placeholder="openai/gpt-oss-120b"
                                    />
                                )}
                            </FormField>
                            <FormField
                                label="API key"
                                htmlFor="ai-key"
                                description={aiHasKey ? 'Leave blank to keep the saved key.' : 'Leave blank only for endpoints that need no key (e.g. local Ollama).'}
                            >
                                <Input
                                    id="ai-key"
                                    type="password"
                                    value={aiForm.apiKey}
                                    onChange={(e) => setAiForm((f) => ({...f, apiKey: e.target.value}))}
                                    placeholder={aiHasKey ? '••••••••' : 'sk-...'}
                                />
                            </FormField>
                            <div className="flex flex-wrap items-center gap-2">
                                <Button
                                    variant="outline"
                                    size="sm"
                                    disabled={aiFetchingModels || !aiForm.baseUrl}
                                    loading={aiFetchingModels}
                                    onClick={fetchAiModels}
                                >
                                    {!aiFetchingModels && <RefreshCwIcon />}
                                    {aiFetchingModels ? 'Fetching…' : 'Fetch models'}
                                </Button>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    disabled={aiTesting || !aiForm.baseUrl || !aiForm.model}
                                    loading={aiTesting}
                                    onClick={testAiConnection}
                                >
                                    {!aiTesting && <PlugZapIcon />}
                                    {aiTesting ? 'Testing…' : 'Test connection'}
                                </Button>
                                {aiTestResult && (
                                    <span className="inline-flex items-center gap-1.5">
                                        <StatusBadge status={aiTestResult.ok ? 'success' : 'failure'} />
                                        {!aiTestResult.ok && <span className="text-xs text-destructive">{aiTestResult.message}</span>}
                                    </span>
                                )}
                            </div>
                        </div>
                    </SheetBody>
                    <SheetFooter>
                        <Button onClick={saveAiProfile} disabled={aiSaving || !aiForm.name || !aiForm.baseUrl} loading={aiSaving}>
                            {aiSaving ? 'Saving…' : 'Save'}
                        </Button>
                    </SheetFooter>
                </SheetContent>
            </Sheet>

            <Dialog open={aiUsageProfile !== null} onOpenChange={(open) => !open && setAiUsageProfile(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>AI usage: {aiUsageProfile?.name}</DialogTitle>
                        <DialogDescription>
                            What Ticketsmith has spent against this endpoint, plus the provider's current rate-limit snapshot.
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-5 py-2">
                        {aiUsageLoading && <LoadingPlaceholder label="Checking…" />}
                        {!aiUsageLoading && aiUsageError && (
                            <p className="text-sm text-destructive">Couldn't fetch usage: {aiUsageError}</p>
                        )}
                        {!aiUsageLoading && !aiUsageError && aiUsage && (
                            <>
                                <div className="grid gap-2">
                                    <h4 className="text-sm font-semibold">Recorded by Ticketsmith</h4>
                                    <RecordedUsageTable summary={aiUsage.recorded} />
                                    <p className="text-xs text-muted-foreground">
                                        Counted from each AI response this app makes. Requests from other apps or the provider's own console aren't included.
                                    </p>
                                </div>
                                <div className="grid gap-3">
                                    <h4 className="text-sm font-semibold">Provider rate limits</h4>
                                    {aiUsage.providerError && (
                                        <p className="text-sm text-destructive">Couldn't reach the provider: {aiUsage.providerError}</p>
                                    )}
                                    {!aiUsage.providerError && !aiUsage.provider.supported && (
                                        <p className="text-sm text-muted-foreground">
                                            This provider doesn't report rate-limit headers.
                                        </p>
                                    )}
                                    {!aiUsage.providerError && aiUsage.provider.supported && (
                                        <>
                                            {aiUsage.provider.requestsLimit > 0 && (
                                                <UsageBar
                                                    label="Requests"
                                                    remaining={aiUsage.provider.requestsRemaining}
                                                    limit={aiUsage.provider.requestsLimit}
                                                    windowSeconds={aiUsage.provider.requestsWindowSeconds}
                                                />
                                            )}
                                            {aiUsage.provider.tokensLimit > 0 && (
                                                <UsageBar
                                                    label="Tokens"
                                                    remaining={aiUsage.provider.tokensRemaining}
                                                    limit={aiUsage.provider.tokensLimit}
                                                    windowSeconds={aiUsage.provider.tokensWindowSeconds}
                                                />
                                            )}
                                            <p className="text-xs text-muted-foreground">
                                                Short rolling-window throttles from the provider's response headers. They refill within seconds to minutes, so they reflect burst headroom, not how much you've consumed overall.
                                            </p>
                                        </>
                                    )}
                                </div>
                            </>
                        )}
                    </div>
                </DialogContent>
            </Dialog>
        </div>
    )
}
