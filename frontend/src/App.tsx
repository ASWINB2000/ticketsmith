import {useEffect, useState} from 'react'
import {Sidebar, type ScreenKey} from '@/components/Layout/Sidebar'
import {Toaster} from '@/components/ui/sonner'
import {ConnectionsProvider} from '@/lib/connections'
import {api} from '@/lib/api'
import {Connect} from '@/screens/Connect'
import {Generate} from '@/screens/Generate'
import {Templates} from '@/screens/Templates'
import {Logs} from '@/screens/Logs'
import {Help} from '@/screens/Help'

function App() {
    const [screen, setScreen] = useState<ScreenKey>('generate')

    // The native window starts hidden (see main.go) so it never shows a
    // blank/stale frame before this first render lands; reveal it now.
    useEffect(() => {
        api.windowReady()
    }, [])

    return (
        <ConnectionsProvider>
            <div className="flex h-screen bg-background text-foreground">
                <Sidebar active={screen} onChange={setScreen} />
                <main className="flex-1 overflow-y-auto">
                    {/* Generate stays mounted (just hidden) so switching tabs never wipes
                        its in-progress state — configured destination, notes, generated
                        preview. The other screens are cheap to reload and stay
                        conditionally-mounted so any open Sheet on them still closes
                        on navigation instead of floating over the next screen (Sheets
                        render via a portal, so hiding an ancestor alone wouldn't hide them). */}
                    <div className={screen === 'generate' ? 'contents' : 'hidden'}>
                        <Generate active={screen === 'generate'} />
                    </div>
                    {screen === 'connect' && <Connect />}
                    {screen === 'templates' && <Templates />}
                    {screen === 'logs' && <Logs />}
                    {screen === 'help' && <Help />}
                </main>
                <Toaster />
            </div>
        </ConnectionsProvider>
    )
}

export default App
