import {useState} from 'react'
import {NavBar, type ScreenKey} from '@/components/Layout/NavBar'
import {Toaster} from '@/components/ui/sonner'
import {Connect} from '@/screens/Connect'
import {Generate} from '@/screens/Generate'
import {Templates} from '@/screens/Templates'
import {Logs} from '@/screens/Logs'

function App() {
    const [screen, setScreen] = useState<ScreenKey>('generate')

    return (
        <div className="min-h-screen bg-background text-foreground">
            <NavBar active={screen} onChange={setScreen} />
            {screen === 'connect' && <Connect />}
            {screen === 'generate' && <Generate />}
            {screen === 'templates' && <Templates />}
            {screen === 'logs' && <Logs />}
            <Toaster />
        </div>
    )
}

export default App
