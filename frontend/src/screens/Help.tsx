import {useEffect, useState, type ReactNode} from 'react'
import {api} from '@/lib/api'
import {logs} from '../../wailsjs/go/models'
import {Card, CardContent} from '@/components/ui/card'
import {PageHeader} from '@/components/Layout/PageHeader'
import {CircleHelpIcon, CheckCircle2Icon, CircleIcon, ChevronDownIcon} from 'lucide-react'

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
    children,
}: {
    done: boolean
    title: string
    status: string
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
                description={`${done} / 4 set up —> expand any step for details.`}
            />
            <div className="grid gap-6 p-8">
                <Card>
                    <CardContent className="divide-y">
                        <ChecklistItem
                            done={state.connections > 0}
                            title="Connect a tracker"
                            status={state.connections > 0 ? `${state.connections} connected` : 'Not connected yet — go to Connect'}
                        >
                            <p>Add your tracker's base URL, e.g. <span className="font-medium text-foreground">https://yourcompany.openproject.com</span>, plus an API token, then click Test.</p>
                            <p>OpenProject tokens: My account → Access tokens → API.</p>
                        </ChecklistItem>

                        <ChecklistItem
                            done={state.aiConfigured}
                            title="Set up the AI provider"
                            status={state.aiConfigured ? 'API key saved' : 'Not configured yet — go to Connect'}
                        >
                            <p>Any OpenAI-compatible endpoint works. OpenAI, Groq, or a local server.</p>
                            <p>Add the base URL, key, and model, then click Test connection.</p>
                        </ChecklistItem>

                        <ChecklistItem
                            done={state.templates > 0}
                            title="Create a template"
                            status={state.templates > 0 ? `${state.templates} template${state.templates === 1 ? '' : 's'}` : 'Not created yet — go to Templates'}
                        >
                            <p>The tracker type name must exactly match a type in your tracker (e.g. "Bug").</p>
                            <p>AI instructions tell the model how to write the subject and description.</p>
                        </ChecklistItem>

                        <ChecklistItem
                            done={state.tickets > 0}
                            title="Generate a ticket"
                            status={state.tickets > 0 ? 'Ticket filed' : 'Ready when you are — go to Generate'}
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
