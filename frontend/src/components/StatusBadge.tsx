import {cn} from '@/lib/utils'

const STYLES: Record<string, string> = {
    success: 'bg-emerald-500/12 text-emerald-600',
    failure: 'bg-destructive/10 text-destructive',
}

export function StatusBadge({status}: { status: string }) {
    const style = STYLES[status] ?? 'bg-muted text-muted-foreground'
    return (
        <span className={cn('inline-flex h-5 w-fit items-center gap-1.5 rounded-full px-2 text-xs font-medium capitalize', style)}>
            <span className={cn('size-1.5 rounded-full', status === 'success' ? 'bg-emerald-500' : status === 'failure' ? 'bg-destructive' : 'bg-muted-foreground')} />
            {status}
        </span>
    )
}
