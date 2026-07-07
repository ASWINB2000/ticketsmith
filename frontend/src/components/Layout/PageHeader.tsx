import type {ComponentType, ReactNode} from 'react'

interface PageHeaderProps {
    icon?: ComponentType<{ className?: string }>
    title: string
    description?: string
    actions?: ReactNode
}

export function PageHeader({icon: Icon, title, description, actions}: PageHeaderProps) {
    return (
        <div className="flex items-start justify-between gap-4 border-b bg-background/80 px-8 py-5 backdrop-blur-sm">
            <div className="flex items-start gap-3">
                {Icon && (
                    <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-accent text-accent-foreground">
                        <Icon className="size-4.5" />
                    </div>
                )}
                <div>
                    <h1 className="font-heading text-lg font-semibold tracking-tight">{title}</h1>
                    {description && <p className="mt-0.5 text-sm text-muted-foreground">{description}</p>}
                </div>
            </div>
            {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
        </div>
    )
}
