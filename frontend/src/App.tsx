import {useEffect, useState} from 'react'
import {Sidebar, type ScreenKey} from '@/components/Layout/Sidebar'
import {Toaster} from '@/components/ui/sonner'
import {ConnectionsProvider} from '@/lib/connections'
import {QuickCapture} from '@/components/QuickCapture'
import {AnnouncementBanner} from '@/components/AnnouncementBanner'
import {api} from '@/lib/api'
import type {NotesPrefill} from '@/lib/notesPrefill'
import {Connect} from '@/screens/Connect'
import {Generate} from '@/screens/Generate'
import {Notes} from '@/screens/Notes'
import {Templates} from '@/screens/Templates'
import {Logs} from '@/screens/Logs'
import {Help} from '@/screens/Help'

function App() {
    const [screen, setScreen] = useState<ScreenKey>('generate')
    const [notesPrefill, setNotesPrefill] = useState<NotesPrefill | null>(null)
    // Bumped when the global-hotkey quick capture saves a note, so an
    // already-open Notes board remounts and shows the new note immediately.
    const [notesVersion, setNotesVersion] = useState(0)

    // The native window starts hidden (see main.go) so it never shows a
    // blank/stale frame before this first render lands; reveal it now.
    useEffect(() => {
        api.windowReady()
    }, [])

    const convertNoteToGenerate = (prefill: NotesPrefill) => {
        setNotesPrefill(prefill)
        setScreen('generate')
    }

    return (
        <ConnectionsProvider>
            <div className="flex h-screen flex-col bg-background text-foreground">
                <AnnouncementBanner />
                <div className="flex flex-1 overflow-hidden">
                    <Sidebar active={screen} onChange={setScreen} />
                    <main className="flex-1 overflow-y-auto">
                        {/* Generate stays mounted (just hidden) so switching tabs never wipes
                            its in-progress state — configured destination, notes, generated
                            preview. The other screens are cheap to reload and stay
                            conditionally-mounted so any open Sheet on them still closes
                            on navigation instead of floating over the next screen (Sheets
                            render via a portal, so hiding an ancestor alone wouldn't hide them). */}
                        <div className={screen === 'generate' ? 'contents' : 'hidden'}>
                            <Generate
                                active={screen === 'generate'}
                                prefill={notesPrefill}
                                onPrefillConsumed={() => setNotesPrefill(null)}
                            />
                        </div>
                        {screen === 'connect' && <Connect />}
                        {screen === 'notes' && <Notes key={notesVersion} onConvertToGenerate={convertNoteToGenerate} />}
                        {screen === 'templates' && <Templates />}
                        {screen === 'logs' && <Logs />}
                        {screen === 'help' && <Help />}
                    </main>
                </div>
                <QuickCapture onSaved={() => setNotesVersion((v) => v + 1)} />
                <Toaster />
            </div>
        </ConnectionsProvider>
    )
}

export default App
