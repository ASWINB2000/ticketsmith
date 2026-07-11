// Shared shape for the Notes -> Generate handoff (docs/NOTES_PLAN.md §5).
// Ephemeral frontend routing state only — nothing here is persisted.
export interface NotesPrefill {
    content: string
    sourceNoteIds: string[]
}
