import {Badge} from '@/components/ui/badge'

export function StatusBadge({status}: { status: string }) {
    const variant = status === 'success' ? 'secondary' : status === 'failure' ? 'destructive' : 'outline'
    return <Badge variant={variant}>{status}</Badge>
}
