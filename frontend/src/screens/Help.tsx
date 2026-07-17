import {useEffect, useState, type ReactNode} from 'react'
import {api} from '@/lib/api'
import {logs} from '../../wailsjs/go/models'
import {Card, CardContent} from '@/components/ui/card'
import {Tabs, TabsList, TabsTrigger, TabsContent} from '@/components/ui/tabs'
import {PageHeader} from '@/components/Layout/PageHeader'
import {OpenAIIcon, GroqIcon, GitHubIcon, OpenProjectIcon, JiraIcon, AzureDevOpsIcon} from '@/components/BrandIcons'
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
        value: 'github',
        label: 'GitHub Models',
        icon: GitHubIcon,
        steps: [
            <>At <Code>github.com/settings/tokens</Code>, generate a fine-grained personal access token with the <span className="font-medium text-foreground">Models</span> permission set to read-only (no repo or org access needed).</>,
            <>In TicketSmith's AI provider card: Base URL <Code>https://models.github.ai/inference</Code>, Model e.g. <Code>gpt-4o</Code> (Llama, Phi, DeepSeek, and Mistral variants are also available), and paste the token into API key.</>,
            <>GitHub's inference endpoint doesn't implement the model-listing call TicketSmith uses, so skip <span className="font-medium text-foreground">Fetch models</span> and type the model name in directly instead.</>,
            <>TicketSmith automatically falls back to a minimal chat request when model listing isn't supported, so click <span className="font-medium text-foreground">Test connection</span> instead to validate the token and model.</>,
            <>This is the same underlying account as signing in to OpenAI "via GitHub," but it's a distinct, free, separate API from OpenAI's own, with its own token and rate limits.</>,
        ],
    },
    {
        value: 'openai',
        label: 'OpenAI',
        icon: OpenAIIcon,
        steps: [
            <>Sign in at <Code>platform.openai.com</Code>. Signing in "via GitHub" is just a login method, so it lands on the same account and the steps below are unaffected.</>,
            <>Add a payment method under <Code>Settings → Billing</Code>. OpenAI won't let a key make requests without one on file.</>,
            <>Go to <Code>Settings → API keys</Code> (or open <Code>{'platform.openai.com/api-keys'}</Code> directly) and click <span className="font-medium text-foreground">Create new secret key</span>.</>,
            <>Copy the key immediately. OpenAI shows it once and never displays it again.</>,
            <>In TicketSmith's AI provider card: Base URL <Code>https://api.openai.com/v1</Code>, Model e.g. <Code>gpt-4.1-mini</Code>, and paste the key into API key. Click <span className="font-medium text-foreground">Fetch models</span> to confirm it works, then Save.</>,
        ],
    },
    {
        value: 'groq',
        label: 'Groq',
        icon: GroqIcon,
        steps: [
            <>Sign in at <Code>console.groq.com</Code>.</>,
            <>Go to <Code>API Keys</Code> in the left nav (or open <Code>{'console.groq.com/keys'}</Code> directly) and click <span className="font-medium text-foreground">Create API Key</span>.</>,
            <>Copy the key. Like most providers, Groq only shows the full value once.</>,
            <>In TicketSmith's AI provider card: Base URL <Code>https://api.groq.com/openai/v1</Code>, Model e.g. <Code>llama-3.3-70b-versatile</Code>, and paste the key into API key. Click <span className="font-medium text-foreground">Fetch models</span> to confirm it works, then Save.</>,
            <>Groq's free tier is generous and fast, a good default if you don't already have OpenAI billing set up.</>,
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
        api.aiSettings.get().then((a) => setState((s) => ({...s, aiConfigured: a.hasKey}))).catch(() => {})
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
                            <p>Any OpenAI-compatible endpoint works. OpenAI, Groq, or a local server.</p>
                            <p>Add the base URL, key, and model, then click Test connection.</p>
                            <ProviderGuideTabs guides={PROVIDER_GUIDES} />
                        </ChecklistItem>

                        <ChecklistItem
                            done={state.templates > 0}
                            title="Create a template"
                            status={state.templates > 0 ? `${state.templates} template${state.templates === 1 ? '' : 's'}` : 'Not created yet, go to Templates'}
                        >
                            <p>The tracker type name must exactly match a type in your tracker (e.g. "Bug").</p>
                            <p>AI instructions tell the model how to write the subject and description.</p>
                            <p>Once you've filed a few tickets, click <span className="font-medium text-foreground">Tune</span> on a template — TicketSmith studies the manual edits you made before filing and suggests improved instructions.</p>
                        </ChecklistItem>

                        <ChecklistItem
                            done={state.tickets > 0}
                            title="Generate a ticket"
                            status={state.tickets > 0 ? 'Ticket filed' : 'Ready when you are, go to Generate'}
                        >
                            <p>Pick a connection, project, and template, then paste your notes and click Generate.</p>
                            <p>Review the AI's output, edit anything, then click Create ticket.</p>
                            <p>Tip: press <Code>Ctrl+T</Code> from anywhere — even outside TicketSmith — to pop a quick-capture window that saves straight to your Notes board. Note it's global, so it overrides Ctrl+T elsewhere (e.g. a browser's new-tab shortcut) while TicketSmith is running.</p>
                        </ChecklistItem>
                    </CardContent>
                </Card>
            </div>
        </div>
    )
}
