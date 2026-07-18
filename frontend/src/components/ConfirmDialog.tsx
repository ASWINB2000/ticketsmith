import type {ReactElement} from 'react'
import {
    Dialog,
    DialogClose,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from '@/components/ui/dialog'
import {Button} from '@/components/ui/button'

interface ConfirmDialogProps {
    trigger: ReactElement
    title: string
    description?: string
    confirmLabel?: string
    destructive?: boolean
    onConfirm: () => void
}

export function ConfirmDialog({
    trigger,
    title,
    description,
    confirmLabel = 'Confirm',
    destructive = false,
    onConfirm,
}: ConfirmDialogProps) {
    return (
        <Dialog>
            <DialogTrigger render={trigger} />
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>{title}</DialogTitle>
                    {description && <DialogDescription>{description}</DialogDescription>}
                </DialogHeader>
                <DialogFooter>
                    <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
                    <DialogClose render={<Button variant={destructive ? 'destructive' : 'default'} onClick={onConfirm} />}>
                        {confirmLabel}
                    </DialogClose>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    )
}
