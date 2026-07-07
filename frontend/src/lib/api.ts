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
    },
    aiSettings: {
        get: Go.GetAISettings,
        save: Go.SaveAISettings,
        listModels: Go.ListAIModels,
        test: Go.TestAISettings,
    },
    templates: {
        list: Go.ListTemplates,
        get: Go.GetTemplate,
        create: Go.CreateTemplate,
        update: Go.UpdateTemplate,
        remove: Go.DeleteTemplate,
    },
    generate: {
        run: Go.GenerateTicket,
        create: Go.CreateTicket,
        getDestination: Go.GetGenerateDestination,
        saveDestination: Go.SaveGenerateDestination,
    },
    logs: {
        list: Go.ListLogs,
        get: Go.GetLog,
    },
    windowReady: Go.WindowReady,
}
