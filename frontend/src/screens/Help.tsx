import {useEffect, useState, type ReactNode} from 'react'
import {api} from '@/lib/api'
import {logs} from '../../wailsjs/go/models'
import {Card, CardContent} from '@/components/ui/card'
import {Tabs, TabsList, TabsTrigger, TabsContent} from '@/components/ui/tabs'
import {PageHeader} from '@/components/Layout/PageHeader'
import {OpenAIIcon, GroqIcon, GeminiIcon, OpenProjectIcon, JiraIcon, AzureDevOpsIcon} from '@/components/BrandIcons'
import {cn} from '@/lib/utils'
import {CircleHelpIcon, CheckCircle2Icon, CircleIcon, ChevronDownIcon} from 'lucide-react'
import type {ComponentType} from 'react'

function Code({children}: { children: ReactNode }) {
    return <code className="break-all rounded bg-muted px-1 py-0.5 font-mono text-[0.8em] text-foreground">{children}</code>
}

interface ProviderGuide {
    value: string
    label: string
    icon: ComponentType<{ className?: string }>
    tint?: string
    steps: ReactNode[]
}

const PROVIDER_GUIDES: ProviderGuide[] = [
    {
        value: 'gemini',
        label: 'Google Gemini',
        icon: GeminiIcon,
        steps: [
            <>Sign in at <Code>aistudio.google.com</Code> and click <span className="font-medium text-foreground">Get API key</span>, then <span className="font-medium text-foreground">Create API key</span>. Free, no credit card.</>,
            <>Add an AI profile with the <span className="font-medium text-foreground">Gemini</span> quick-fill and paste the key.</>,
            <>Click <span className="font-medium text-foreground">Fetch models</span> and pick the newest Flash model, e.g. <Code>gemini-3.5-flash</Code>.</>,
            <>Free keys only cover Flash models. Pro models fail with a 429 error.</>,
            <>Google may train on free-tier prompts, so keep sensitive details out.</>,
        ],
    },
    {
        value: 'openai',
        label: 'OpenAI',
        icon: OpenAIIcon,
        steps: [
            <>Sign in at <Code>platform.openai.com</Code> and add a payment method under <Code>Settings → Billing</Code>. Keys don't work without one.</>,
            <>Open <Code>{'platform.openai.com/api-keys'}</Code> and click <span className="font-medium text-foreground">Create new secret key</span>. Copy it right away, it's shown only once.</>,
            <>Add an AI profile with the <span className="font-medium text-foreground">OpenAI</span> quick-fill and paste the key.</>,
        ],
    },
    {
        value: 'groq',
        label: 'Groq',
        icon: GroqIcon,
        steps: [
            <>Sign in at <Code>console.groq.com</Code>, open <Code>{'console.groq.com/keys'}</Code>, and click <span className="font-medium text-foreground">Create API Key</span>.</>,
            <>Copy the key. Groq shows it only once.</>,
            <>Add an AI profile with the <span className="font-medium text-foreground">Groq</span> quick-fill and paste the key.</>,
            <>Keep the default model <Code>openai/gpt-oss-120b</Code>. It's the strongest writer Groq hosts.</>,
        ],
    },
]

const TRACKER_GUIDES: ProviderGuide[] = [
    {
        value: 'openproject',
        label: 'OpenProject',
        icon: OpenProjectIcon,
        tint: 'text-sky-700 bg-sky-600/10',
        steps: [
            <>Click your user icon (top right) → <span className="font-medium text-foreground">Account settings</span>, then <span className="font-medium text-foreground">Access tokens</span> in the left nav.</>,
            <>Next to <span className="font-medium text-foreground">API</span>, click <span className="font-medium text-foreground">Generate</span> (or <span className="font-medium text-foreground">Reset</span> if one already exists; resetting invalidates the old one).</>,
            <>Copy the token immediately. OpenProject shows it once and can't display it again.</>,
            <>In TicketSmith's Add connection sheet: Base URL is the instance root, e.g. <Code>https://yourcompany.openproject.com</Code> (no trailing path), and paste the token into API token.</>,
            <>Only one API token can exist per user at a time. Generating a new one retires the previous one everywhere it's used.</>,
        ],
    },
    {
        value: 'jira',
        label: 'Jira',
        icon: JiraIcon,
        tint: 'text-blue-700 bg-blue-600/10',
        steps: [
            <>Sign in, then open <Code>id.atlassian.com/manage-profile/security/api-tokens</Code> directly (it's under your Atlassian account, not a specific Jira site).</>,
            <>Click <span className="font-medium text-foreground">Create API token</span>, give it a name, and set an expiry (up to 365 days).</>,
            <>Copy the token immediately. Like OpenProject, Atlassian shows it once and won't store or redisplay the value.</>,
            <>Jira authenticates with your account email plus this token (basic auth), not the token alone.</>,
        ],
    },
    {
        value: 'azuredevops',
        label: 'Azure DevOps',
        icon: AzureDevOpsIcon,
        tint: 'text-sky-700 bg-sky-600/10',
        steps: [
            <>Sign in to your organization at <Code>dev.azure.com/{'{organization}'}</Code>, then click the user-settings icon (top right) → <span className="font-medium text-foreground">Personal access tokens</span>.</>,
            <>Click <span className="font-medium text-foreground">+ New Token</span>, name it, pick the organization, and set an expiration.</>,
            <>Under Scopes, choose <span className="font-medium text-foreground">Custom defined</span> and grant <span className="font-medium text-foreground">Work Items (Read &amp; Write)</span>. Avoid Full access for a single-purpose integration.</>,
            <>Copy the token immediately. Azure DevOps displays it only at creation time and it can't be retrieved afterward.</>,
        ],
    },
]

function ProviderGuideTabs({guides}: { guides: ProviderGuide[] }) {
    return (
        <Tabs defaultValue={guides[0].value} className="mt-1">
            <TabsList>
                {guides.map((p) => (
                    <TabsTrigger key={p.value} value={p.value} className="gap-1.5">
                        <p.icon className="size-3.5" />
                        {p.label}
                    </TabsTrigger>
                ))}
            </TabsList>
            {guides.map((p) => (
                <TabsContent key={p.value} value={p.value}>
                    <ol className="grid list-decimal gap-1.5 pl-4">
                        {p.steps.map((step, i) => <li key={i}>{step}</li>)}
                    </ol>
                </TabsContent>
            ))}
        </Tabs>
    )
}

interface SetupState {
    connections: number
    aiConfigured: boolean
    templates: number
    tickets: number
}

function ChecklistItem({
    done,
    title,
    status,
    icons,
    children,
}: {
    done: boolean
    title: string
    status: string
    icons?: Pick<ProviderGuide, 'icon' | 'tint' | 'label'>[]
    children: ReactNode
}) {
    return (
        <details className="group py-4 first:pt-0 last:pb-0">
            <summary className="flex cursor-pointer list-none items-center gap-3 [&::-webkit-details-marker]:hidden">
                {done ? (
                    <CheckCircle2Icon className="size-5 shrink-0 text-emerald-600" />
                ) : (
                    <CircleIcon className="size-5 shrink-0 text-muted-foreground/40" />
                )}
                <div className="flex-1">
                    <p className="text-sm font-medium">{title}</p>
                    <p className="text-xs text-muted-foreground">{status}</p>
                </div>
                {icons && (
                    <div className="flex items-center gap-2.5">
                        {icons.map(({icon: Icon, tint, label}) => (
                            <div
                                key={label}
                                title={label}
                                className={cn('flex size-10 items-center justify-center rounded-lg', tint ?? 'bg-muted text-foreground')}
                            >
                                <Icon className="size-5" />
                            </div>
                        ))}
                    </div>
                )}
                <ChevronDownIcon className="size-4 shrink-0 text-muted-foreground/60 transition-transform group-open:rotate-180" />
            </summary>
            <div className="mt-3 ml-8 grid gap-1.5 text-xs text-muted-foreground">{children}</div>
        </details>
    )
}

export function Help() {
    const [state, setState] = useState<SetupState>({connections: 0, aiConfigured: false, templates: 0, tickets: 0})

    useEffect(() => {
        api.connections.list().then((c) => setState((s) => ({...s, connections: c.length}))).catch(() => {})
        api.aiProfiles.list().then((ps) => setState((s) => ({...s, aiConfigured: ps.length > 0}))).catch(() => {})
        api.templates.list().then((t) => setState((s) => ({...s, templates: t.length}))).catch(() => {})
        api.logs.list(new logs.Filter({action: 'create', status: 'success'})).then((l) => setState((s) => ({...s, tickets: l.length}))).catch(() => {})
    }, [])

    const done = [state.connections > 0, state.aiConfigured, state.templates > 0, state.tickets > 0].filter(Boolean).length

    return (
        <div className="flex flex-col">
            <PageHeader
                icon={CircleHelpIcon}
                title="Help"
                description={`${done} / 4 set up. Expand any step for details.`}
            />
            <div className="grid gap-6 p-8">
                <Card>
                    <CardContent className="divide-y">
                        <ChecklistItem
                            done={state.connections > 0}
                            title="Connect a tracker"
                            status={state.connections > 0 ? `${state.connections} connected` : 'Not connected yet, go to Connect'}
                            icons={TRACKER_GUIDES}
                        >
                            <p>Add your tracker's base URL and an API token, then click Test. OpenProject is supported today; Jira and Azure DevOps are planned.</p>
                            <ProviderGuideTabs guides={TRACKER_GUIDES} />
                        </ChecklistItem>

                        <ChecklistItem
                            done={state.aiConfigured}
                            title="Set up the AI provider"
                            status={state.aiConfigured ? 'API key saved' : 'Not configured yet, go to Connect'}
                            icons={PROVIDER_GUIDES}
                        >
                            <p>Save each provider as a profile, then pick which one is active. Any OpenAI-compatible endpoint works.</p>
                            <ProviderGuideTabs guides={PROVIDER_GUIDES} />
                        </ChecklistItem>

                        <ChecklistItem
                            done={state.templates > 0}
                            title="Create a template"
                            status={state.templates > 0 ? `${state.templates} template${state.templates === 1 ? '' : 's'}` : 'Not created yet, go to Templates'}
                        >
                            <p>The tracker type name must exactly match a type in your tracker (e.g. "Bug").</p>
                            <p>AI instructions tell the model how to write the subject and description.</p>
                        </ChecklistItem>

                        <ChecklistItem
                            done={state.tickets > 0}
                            title="Generate a ticket"
                            status={state.tickets > 0 ? 'Ticket filed' : 'Ready when you are, go to Generate'}
                        >
                            <p>Pick a connection, project, and template, then paste your notes and click Generate.</p>
                            <p>Review the AI's output, edit anything, then click Create ticket.</p>
                        </ChecklistItem>
                    </CardContent>
                </Card>
            </div>
        </div>
    )
}
