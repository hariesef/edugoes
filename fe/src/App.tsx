import { useState } from 'react'
import './App.css'
import { Sidebar } from './components/Sidebar'
import { ToolRegistration } from './pages/ToolRegistration'
import { ToolLaunch } from './pages/ToolLaunch'

type MenuKey = 'tool-registration' | 'tool-launch'

function App() {
  const [active, setActive] = useState<MenuKey>('tool-registration')

  return (
    <div className="app">
      <Sidebar
        items={[
          { key: 'tool-registration', label: 'Tool Registration' },
          { key: 'tool-launch', label: 'Tool Launch' },
        ]}
        active={active}
        onSelect={(key) => setActive(key as MenuKey)}
      />
      <main className="main">
        {active === 'tool-registration' && <ToolRegistration />}
        {active === 'tool-launch' && <ToolLaunch />}
      </main>
    </div>
  )
}

export default App
