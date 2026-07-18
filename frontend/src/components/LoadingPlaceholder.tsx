import {Loader2Icon} from 'lucide-react'
import {cn} from '@/lib/utils'

// Shared block-level loading state for panels/dialogs waiting on an async
// result (AI analysis, usage lookups, release notes, etc.) — replaces plain
// "Loading…" text with a consistent spinner + message treatment.
export function LoadingPlaceholder({label, className}: {label: string; className?: string}) {
    return (
        <div className={cn('flex flex-col items-center justify-center gap-2.5 rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground', className)}>
            <Loader2Icon className="size-5 animate-spin text-muted-foreground/70" />
            {label}
        </div>
    )
}
