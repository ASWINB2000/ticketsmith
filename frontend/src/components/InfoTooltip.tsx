import {Tooltip, TooltipContent, TooltipTrigger} from '@/components/ui/tooltip'
import {InfoIcon} from 'lucide-react'

export function InfoTooltip({children}: { children: React.ReactNode }) {
    return (
        <Tooltip>
            <TooltipTrigger
                className="inline-flex text-muted-foreground outline-none hover:text-foreground focus-visible:text-foreground"
                aria-label="More info"
            >
                <InfoIcon className="size-3.5" />
            </TooltipTrigger>
            <TooltipContent side="top">{children}</TooltipContent>
        </Tooltip>
    )
}
