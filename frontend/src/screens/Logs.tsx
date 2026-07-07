import {useEffect, useState} from 'react'
import {toast} from 'sonner'
import {logs, connections} from '../../wailsjs/go/models'
import {BrowserOpenURL} from '../../wailsjs/runtime/runtime'
import {api} from '@/lib/api'
import {Card, CardContent} from '@/components/ui/card'
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select'
import {Sheet, SheetContent, SheetHeader, SheetTitle, SheetBody} from '@/components/ui/sheet'
import {DataTable, type DataTableColumn} from '@/components/DataTable'
import {StatusBadge} from '@/components/StatusBadge'
import {FormField} from '@/components/FormField'
import {PageHeader} from '@/components/Layout/PageHeader'
import {ScrollTextIcon} from 'lucide-react'

type LogEntry = logs.LogEntry
type Connection = connections.Connection

const ANY = '__any__'

export function Logs() {
    const [list, setList] = useState<LogEntry[]>([])
    const [conns, setConns] = useState<Connection[]>([])
    const [action, setAction] = useState(ANY)
    const [status, setStatus] = useState(ANY)
    const [connectionId, setConnectionId] = useState(ANY)
    const [selected, setSelected] = useState<LogEntry | null>(null)

    const refresh = () => {
        api.logs.list(new logs.Filter({
            action: action === ANY ? '' : action,
            status: status === ANY ? '' : status,
            connectionId: connectionId === ANY ? '' : connectionId,
        })).then(setList).catch((err) => toast.error(`Failed to load logs: ${err}`))
    }

    useEffect(() => {
        api.connections.list().then(setConns).catch(() => {})
    }, [])

    useEffect(refresh, [action, status, connectionId])

    const connectionName = (id?: string) => conns.find((c) => c.id === id)?.name ?? id ?? '—'

    const columns: DataTableColumn<LogEntry>[] = [
        {key: 'timestamp', header: 'Time', render: (l) => new Date(l.timestamp).toLocaleString()},
        {key: 'action', header: 'Action', render: (l) => l.action},
        {key: 'connection', header: 'Connection', render: (l) => connectionName(l.connectionId)},
        {key: 'status', header: 'Status', render: (l) => <StatusBadge status={l.status} />},
        {
            key: 'ticket',
            header: 'Ticket',
            render: (l) => l.resultTicketUrl ? (
                <button
                    className="text-primary underline underline-offset-4"
                    onClick={(e) => {
                        e.stopPropagation()
                        BrowserOpenURL(l.resultTicketUrl!)
                    }}
                >
                    {l.resultTicketId}
                </button>
            ) : '—',
        },
    ]

    return (
        <div className="flex flex-col">
            <PageHeader icon={ScrollTextIcon} title="Logs" description="Full audit trail of every generate/create action." />
            <div className="grid gap-4 p-8">
            <Card>
                <CardContent className="grid gap-4">
                    <div className="grid grid-cols-3 gap-4">
                        <FormField label="Action">
                            <Select
                                value={action}
                                onValueChange={(v) => setAction(v as string)}
                                items={{[ANY]: 'Any', generate: 'Generate', create: 'Create'}}
                            >
                                <SelectTrigger><SelectValue /></SelectTrigger>
                                <SelectContent>
                                    <SelectItem value={ANY}>Any</SelectItem>
                                    <SelectItem value="generate">Generate</SelectItem>
                                    <SelectItem value="create">Create</SelectItem>
                                </SelectContent>
                            </Select>
                        </FormField>
                        <FormField label="Status">
                            <Select
                                value={status}
                                onValueChange={(v) => setStatus(v as string)}
                                items={{[ANY]: 'Any', success: 'Success', failure: 'Failure'}}
                            >
                                <SelectTrigger><SelectValue /></SelectTrigger>
                                <SelectContent>
                                    <SelectItem value={ANY}>Any</SelectItem>
                                    <SelectItem value="success">Success</SelectItem>
                                    <SelectItem value="failure">Failure</SelectItem>
                                </SelectContent>
                            </Select>
                        </FormField>
                        <FormField label="Connection">
                            <Select
                                value={connectionId}
                                onValueChange={(v) => setConnectionId(v as string)}
                                items={{[ANY]: 'Any', ...Object.fromEntries(conns.map((c) => [c.id, c.name]))}}
                            >
                                <SelectTrigger><SelectValue /></SelectTrigger>
                                <SelectContent>
                                    <SelectItem value={ANY}>Any</SelectItem>
                                    {conns.map((c) => <SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>)}
                                </SelectContent>
                            </Select>
                        </FormField>
                    </div>
                    <DataTable
                        columns={columns}
                        rows={list}
                        rowKey={(l) => l.id}
                        emptyMessage="No log entries yet."
                        onRowClick={setSelected}
                    />
                </CardContent>
            </Card>

            <Sheet open={!!selected} onOpenChange={(open) => !open && setSelected(null)}>
                <SheetContent>
                    <SheetHeader>
                        <SheetTitle>Log entry — {selected?.action}</SheetTitle>
                    </SheetHeader>
                    <SheetBody>
                        {selected && (
                            <div className="grid gap-4 text-sm">
                                <div>
                                    <div className="mb-1 font-medium">Raw input</div>
                                    <pre className="max-h-64 overflow-auto rounded-lg bg-muted p-2 font-mono text-xs whitespace-pre-wrap">{selected.rawInput || '—'}</pre>
                                </div>
                                <div>
                                    <div className="mb-1 font-medium">Generated content</div>
                                    <pre className="max-h-64 overflow-auto rounded-lg bg-muted p-2 font-mono text-xs whitespace-pre-wrap">{selected.generatedContent || '—'}</pre>
                                </div>
                                <div>
                                    <div className="mb-1 font-medium">Final content</div>
                                    <pre className="max-h-64 overflow-auto rounded-lg bg-muted p-2 font-mono text-xs whitespace-pre-wrap">{selected.finalContent || '—'}</pre>
                                </div>
                                {selected.errorMessage && (
                                    <div>
                                        <div className="mb-1 font-medium text-destructive">Error</div>
                                        <pre className="max-h-64 overflow-auto rounded-lg bg-destructive/10 p-2 font-mono text-xs whitespace-pre-wrap text-destructive">{selected.errorMessage}</pre>
                                    </div>
                                )}
                            </div>
                        )}
                    </SheetBody>
                </SheetContent>
            </Sheet>
            </div>
        </div>
    )
}
