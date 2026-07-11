import {EditorContent, useEditor} from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import TaskList from '@tiptap/extension-task-list'
import TaskItem from '@tiptap/extension-task-item'
import Placeholder from '@tiptap/extension-placeholder'
import {Markdown, type MarkdownStorage} from 'tiptap-markdown'
import {cn} from '@/lib/utils'
import {Button} from '@/components/ui/button'
import {BoldIcon, ItalicIcon, ListIcon, ListOrderedIcon, ListChecksIcon} from 'lucide-react'

// tiptap-markdown doesn't ship the @tiptap/core Storage module augmentation
// its own README examples rely on — declare it here so `editor.storage.markdown`
// type-checks instead of needing an `any` cast at every call site.
declare module '@tiptap/core' {
    interface Storage {
        markdown: MarkdownStorage
    }
}

interface NoteEditorProps {
    // Markdown. Only read on first mount — Tiptap doesn't resync `content` on
    // later prop changes, so callers that need to force fresh content in
    // (e.g. after a save, or a new AI-regenerated draft) remount via `key`.
    content: string
    onChange?: (markdown: string) => void
    onBlur?: (markdown: string) => void
    placeholder?: string
    editable?: boolean
    // Applied to the editable box itself (border/padding/max-height/overflow).
    className?: string
    // Applied to the outer wrapper (toolbar + box). Only needed when a
    // caller has to make the whole thing a flex child that grows/shrinks
    // within a constrained-height flex ancestor (see the merge dialog).
    wrapperClassName?: string
}

export function NoteEditor({content, onChange, onBlur, placeholder, editable = true, className, wrapperClassName}: NoteEditorProps) {
    const editor = useEditor({
        extensions: [
            StarterKit,
            TaskList,
            TaskItem.configure({nested: false}),
            Placeholder.configure({placeholder}),
            Markdown.configure({html: false}),
        ],
        content,
        editable,
        editorProps: {
            attributes: {
                class: 'tiptap-note text-sm',
            },
        },
        onUpdate: ({editor}) => onChange?.(editor.storage.markdown.getMarkdown()),
        onBlur: ({editor}) => onBlur?.(editor.storage.markdown.getMarkdown()),
    })

    if (!editor) return null

    return (
        <div className={cn('flex flex-col gap-2', !editable && 'opacity-50', wrapperClassName)}>
            {editable && (
                <div className="flex shrink-0 gap-1">
                    <Button
                        type="button"
                        size="icon-xs"
                        variant={editor.isActive('bold') ? 'secondary' : 'ghost'}
                        title="Bold"
                        onClick={() => editor.chain().focus().toggleBold().run()}
                    >
                        <BoldIcon />
                    </Button>
                    <Button
                        type="button"
                        size="icon-xs"
                        variant={editor.isActive('italic') ? 'secondary' : 'ghost'}
                        title="Italic"
                        onClick={() => editor.chain().focus().toggleItalic().run()}
                    >
                        <ItalicIcon />
                    </Button>
                    <Button
                        type="button"
                        size="icon-xs"
                        variant={editor.isActive('bulletList') ? 'secondary' : 'ghost'}
                        title="Bullet list"
                        onClick={() => editor.chain().focus().toggleBulletList().run()}
                    >
                        <ListIcon />
                    </Button>
                    <Button
                        type="button"
                        size="icon-xs"
                        variant={editor.isActive('orderedList') ? 'secondary' : 'ghost'}
                        title="Numbered list"
                        onClick={() => editor.chain().focus().toggleOrderedList().run()}
                    >
                        <ListOrderedIcon />
                    </Button>
                    <Button
                        type="button"
                        size="icon-xs"
                        variant={editor.isActive('taskList') ? 'secondary' : 'ghost'}
                        title="Checklist"
                        onClick={() => editor.chain().focus().toggleTaskList().run()}
                    >
                        <ListChecksIcon />
                    </Button>
                </div>
            )}
            <EditorContent
                editor={editor}
                className={cn(
                    'flex-1 min-h-16 overflow-y-auto rounded-lg border border-input bg-transparent px-2.5 py-2 transition-colors focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50',
                    className,
                )}
            />
        </div>
    )
}
