import type {ReactNode} from 'react'
import {cn} from '@/lib/utils'
import {BrowserOpenURL} from '../../wailsjs/runtime/runtime'

// Renders exactly the markdown shapes GitHub's "generate release notes" API
// produces (## headings, * bullet lists, **bold**, [text](url) links,
// `code`, and bare https:// URLs like the "Full Changelog" line, which
// GitHub emits unwrapped rather than as a markdown link) — not a
// general-purpose markdown parser, since that's the only source this ever
// displays.
function renderInline(text: string, keyPrefix: string): ReactNode[] {
    const nodes: ReactNode[] = []
    const pattern = /\*\*(.+?)\*\*|\[([^\]]+)\]\(([^)]+)\)|`([^`]+)`|(https?:\/\/[^\s)]+)/g
    let lastIndex = 0
    let match: RegExpExecArray | null
    let i = 0

    while ((match = pattern.exec(text))) {
        if (match.index > lastIndex) nodes.push(text.slice(lastIndex, match.index))
        const key = `${keyPrefix}-${i++}`
        if (match[1] !== undefined) {
            nodes.push(<strong key={key} className="font-semibold text-foreground">{match[1]}</strong>)
        } else if (match[2] !== undefined) {
            const url = match[3]
            nodes.push(
                <button
                    key={key}
                    type="button"
                    onClick={() => BrowserOpenURL(url)}
                    className="break-all text-primary underline underline-offset-2 hover:text-primary/80"
                >
                    {match[2]}
                </button>,
            )
        } else if (match[4] !== undefined) {
            nodes.push(
                <code key={key} className="break-all rounded bg-muted px-1 py-0.5 font-mono text-[0.8em]">
                    {match[4]}
                </code>,
            )
        } else if (match[5] !== undefined) {
            const url = match[5]
            nodes.push(
                <button
                    key={key}
                    type="button"
                    onClick={() => BrowserOpenURL(url)}
                    className="break-all text-primary underline underline-offset-2 hover:text-primary/80"
                >
                    {url}
                </button>,
            )
        }
        lastIndex = pattern.lastIndex
    }
    if (lastIndex < text.length) nodes.push(text.slice(lastIndex))
    return nodes
}

export function ReleaseNotes({markdown}: { markdown: string }) {
    const blocks: ReactNode[] = []
    let listItems: ReactNode[] = []
    let listKey = 0

    function flushList() {
        if (listItems.length === 0) return
        blocks.push(
            <ul key={`list-${listKey++}`} className="grid gap-1.5">
                {listItems}
            </ul>,
        )
        listItems = []
    }

    markdown.split('\n').forEach((rawLine, i) => {
        const line = rawLine.trim()
        if (!line) {
            flushList()
            return
        }

        const heading = /^(#{1,3})\s+(.*)$/.exec(line)
        if (heading) {
            flushList()
            const level = heading[1].length
            blocks.push(
                <p
                    key={`h-${i}`}
                    className={cn(
                        'font-heading font-semibold text-foreground',
                        level === 1 ? 'text-base' : level === 2 ? 'text-sm' : 'text-xs uppercase tracking-wide text-muted-foreground',
                    )}
                >
                    {renderInline(heading[2], `h-${i}`)}
                </p>,
            )
            return
        }

        const item = /^[*-]\s+(.*)$/.exec(line)
        if (item) {
            listItems.push(
                <li key={`li-${i}`} className="flex gap-2 text-sm text-muted-foreground">
                    <span className="mt-[7px] size-1 shrink-0 rounded-full bg-muted-foreground/50" />
                    <span className="min-w-0 break-words">{renderInline(item[1], `li-${i}`)}</span>
                </li>,
            )
            return
        }

        flushList()
        blocks.push(
            <p key={`p-${i}`} className="break-words text-sm text-muted-foreground">
                {renderInline(line, `p-${i}`)}
            </p>,
        )
    })
    flushList()

    return <div className="grid min-w-0 gap-2">{blocks}</div>
}
