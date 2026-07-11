import type {ReactNode} from 'react'
import {Label} from '@/components/ui/label'
import {cn} from '@/lib/utils'

interface FormFieldProps {
    label: ReactNode
    htmlFor?: string
    description?: string
    error?: string
    required?: boolean
    className?: string
    children: ReactNode
}

export function FormField({label, htmlFor, description, error, required, className, children}: FormFieldProps) {
    return (
        <div className={cn('grid gap-1.5', className)}>
            <Label htmlFor={htmlFor}>
                {label}
                {required && <span className="text-destructive" aria-hidden="true"> *</span>}
            </Label>
            {children}
            {error ? (
                <p className="text-xs text-destructive">{error}</p>
            ) : description ? (
                <p className="text-xs text-muted-foreground">{description}</p>
            ) : null}
        </div>
    )
}
