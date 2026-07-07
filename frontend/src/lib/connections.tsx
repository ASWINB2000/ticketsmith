// Single source of truth for the connections list, shared across screens.
// Generate stays mounted for the app's whole lifetime and Sidebar never
// remounts, so if each screen fetched its own copy, an edit made on the
// Connect screen would only reach them after an app restart. Every mutation
// (create/update/delete) must call refresh() so all subscribers update
// immediately instead.
import {createContext, useCallback, useContext, useEffect, useState, type ReactNode} from 'react'
import {toast} from 'sonner'
import {connections as connectionsModel} from '../../wailsjs/go/models'
import {api} from './api'

type Connection = connectionsModel.Connection

interface ConnectionsContextValue {
    connections: Connection[]
    refresh: () => Promise<void>
}

const ConnectionsContext = createContext<ConnectionsContextValue | null>(null)

export function ConnectionsProvider({children}: { children: ReactNode }) {
    const [connections, setConnections] = useState<Connection[]>([])

    const refresh = useCallback(async () => {
        try {
            setConnections(await api.connections.list())
        } catch (err) {
            toast.error(`Failed to load connections: ${err}`)
        }
    }, [])

    useEffect(() => {
        refresh()
    }, [refresh])

    return <ConnectionsContext.Provider value={{connections, refresh}}>{children}</ConnectionsContext.Provider>
}

export function useConnections() {
    const ctx = useContext(ConnectionsContext)
    if (!ctx) throw new Error('useConnections must be used within a ConnectionsProvider')
    return ctx
}
