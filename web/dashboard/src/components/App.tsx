import { useEffect, useState } from 'react'

import { api } from './api'
import Dashboard from './Dashboard'
import Login from './Login'

function App() {
  const [ready, setReady] = useState(false)
  const [authenticated, setAuthenticated] = useState(false)

  useEffect(() => {
    async function init() {
      const params = new URLSearchParams(window.location.search)
      const code = params.get('code')

      if (code) {
        try {
          await api.handleCallback(code)
          window.history.replaceState({}, '', '/')
          setAuthenticated(true)
        } catch {
          // ignore
        }
      } else {
        setAuthenticated(api.isAuthenticated())
      }
      setReady(true)
    }
    init()
  }, [])

  if (!ready) {
    return (
      <div className='flex min-h-screen items-center justify-center'>
        <div className='h-8 w-8 animate-spin rounded-full border-4 border-white/10 border-t-neon' />
      </div>
    )
  }

  return authenticated ? <Dashboard /> : <Login />
}

export default App
