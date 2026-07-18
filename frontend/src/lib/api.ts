// Unified backend API client. Every screen must import { api } from here —
// never from wailsjs/go/main/App directly — so a future headless server mode
// only needs this file's internals to change (Wails IPC -> fetch), with zero
// call-site rework in screens.
import * as Go from '../../wailsjs/go/main/App'

export const api = {
    connections: {
        list: Go.ListConnections,
        create: Go.CreateConnection,
        update: Go.UpdateConnection,
        remove: Go.DeleteConnection,
        test: Go.TestConnection,
    },
    tracker: {
        types: Go.GetTrackerTypes,
        projects: Go.GetTrackerProjects,
        assignees: Go.GetTrackerAssignees,
        priorities: Go.GetTrackerPriorities,
        customFields: Go.GetTrackerCustomFields,
    },
    aiProfiles: {
        list: Go.ListAIProfiles,
        create: Go.CreateAIProfile,
        update: Go.UpdateAIProfile,
        remove: Go.DeleteAIProfile,
        setActive: Go.SetActiveAIProfile,
        listModels: Go.ListAIModels,
        test: Go.TestAIProfile,
        usage: Go.AIUsage,
    },
    templates: {
        list: Go.ListTemplates,
        get: Go.GetTemplate,
        create: Go.CreateTemplate,
        update: Go.UpdateTemplate,
        remove: Go.DeleteTemplate,
        suggestTuning: Go.SuggestTemplateTuning,
    },
    generate: {
        run: Go.GenerateTicket,
        refine: Go.RefineTicket,
        create: Go.CreateTicket,
        getDestination: Go.GetGenerateDestination,
        saveDestination: Go.SaveGenerateDestination,
        pickAttachments: Go.PickAttachments,
        uploadAttachments: Go.UploadAttachments,
        saveClipboardAttachment: Go.SaveClipboardAttachment,
        discardClipboardAttachment: Go.DiscardClipboardAttachment,
        attachmentPreview: Go.GetAttachmentPreview,
    },
    logs: {
        list: Go.ListLogs,
        get: Go.GetLog,
    },
    notes: {
        list: Go.ListNotes,
        create: Go.CreateNote,
        update: Go.UpdateNote,
        remove: Go.DeleteNote,
        merge: Go.MergeNotes,
        confirmMerge: Go.ConfirmMerge,
        quickCaptureShortcut: Go.QuickCaptureShortcut,
    },
    releaseNotes: {
        latest: Go.GetLatestReleaseNotes,
    },
    windowReady: Go.WindowReady,
}
