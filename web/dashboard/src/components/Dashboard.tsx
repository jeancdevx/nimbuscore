import { useCallback, useEffect, useState } from 'react'

import type { User, Workspace } from '../types'
import { api } from './api'
import CreateWorkspace from './CreateWorkspace'
import WorkspaceCard from './WorkspaceCard'

function Dashboard() {
  const [user, setUser] = useState<User | null>(null)
  const [workspaces, setWorkspaces] = useState<Workspace[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const [u, ws] = await Promise.all([api.me.get(), api.workspaces.list()])
      setUser(u)
      setWorkspaces(ws)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async (data: Partial<Workspace>) => {
    const ws = await api.workspaces.create(data)
    setWorkspaces(prev => [ws, ...prev])
    setShowCreate(false)
  }

  const handleDelete = async (id: string) => {
    await api.workspaces.delete(id)
    setWorkspaces(prev => prev.filter(ws => ws.id !== id))
  }

  const handleStart = async (id: string) => {
    await api.workspaces.start(id)
    setWorkspaces(prev =>
      prev.map(ws =>
        ws.id === id ? { ...ws, status: 'building' as const } : ws
      )
    )
  }

  const handleStop = async (id: string) => {
    await api.workspaces.stop(id)
    setWorkspaces(prev =>
      prev.map(ws =>
        ws.id === id ? { ...ws, status: 'stopping' as const } : ws
      )
    )
  }

  const runningCount = workspaces.filter(w => w.status === 'running').length
  const totalVotes = workspaces.length

  if (loading) {
    return (
      <div className='flex min-h-screen items-center justify-center'>
        <div className='h-8 w-8 animate-spin rounded-full border-4 border-white/10 border-t-neon' />
      </div>
    )
  }

  return (
    <div className='min-h-screen'>
      <NavBar user={user} />

      <main className='mx-auto w-full max-w-6xl px-4 pb-16'>
        {error && (
          <div className='mb-6 animate-slide-in rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-sm text-danger'>
            {error}
            <button
              onClick={load}
              className='ml-2 underline hover:text-red-300'
            >
              Retry
            </button>
          </div>
        )}

        <Header />

        <section className='mb-8 grid grid-cols-3 gap-2 sm:gap-4'>
          <div className='stat-card'>
            <div className='font-pixel text-xl tabular text-neon sm:text-3xl'>
              {runningCount}
            </div>
            <div className='stat-card__label'>Running</div>
          </div>
          <div className='stat-card'>
            <div className='font-pixel text-xl tabular text-white sm:text-3xl'>
              {totalVotes}
            </div>
            <div className='stat-card__label'>Total</div>
          </div>
          <div className='stat-card'>
            <div className='font-pixel text-xl tabular text-gold sm:text-3xl'>
              {user?.name?.charAt(0)?.toUpperCase() ?? '?'}
            </div>
            <div className='stat-card__label'>{user?.email ?? 'User'}</div>
          </div>
        </section>

        <div className='mb-6 flex items-center justify-between'>
          <h2 className='font-pixel text-lg sm:text-xl'>WORKSPACES</h2>
          <button
            onClick={() => setShowCreate(true)}
            className='btn btn-neon text-xs tracking-wider sm:text-sm'
          >
            + NEW
          </button>
        </div>

        {workspaces.length === 0 ? (
          <div className='rounded-xl border border-dashed border-white/10 p-16 text-center'>
            <p className='text-sm text-white/40'>No workspaces yet</p>
            <button
              onClick={() => setShowCreate(true)}
              className='mt-3 text-sm text-neon hover:text-neon-dim'
            >
              Create your first workspace
            </button>
          </div>
        ) : (
          <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-3'>
            {workspaces.map(ws => (
              <WorkspaceCard
                key={ws.id}
                workspace={ws}
                onStart={() => handleStart(ws.id)}
                onStop={() => handleStop(ws.id)}
                onDelete={() => handleDelete(ws.id)}
              />
            ))}
          </div>
        )}
      </main>

      {showCreate && (
        <CreateWorkspace
          onSubmit={handleCreate}
          onClose={() => setShowCreate(false)}
        />
      )}
    </div>
  )
}

function Header() {
  return (
    <header className='pt-8 pb-6 text-center'>
      <p className='font-pixel text-[10px] tracking-[0.3em] text-neon sm:text-xs'>
        DEV INFRASTRUCTURE · ON-DEMAND
      </p>
      <h1 className='font-pixel text-3xl leading-tight sm:text-5xl lg:text-6xl'>
        NIMBUS<span className='text-neon'>CORE</span>
      </h1>
    </header>
  )
}

function NavBar({ user }: { user: User | null }) {
  const handleLogout = () => {
    api.clearAuth()
    window.location.href = '/login'
  }

  const isAdmin = window.location.pathname === '/admin'

  return (
    <header className='border-b border-white/5 bg-pitch-900/50 backdrop-blur'>
      <div className='mx-auto flex max-w-6xl items-center justify-between px-4 py-3'>
        <div className='flex items-center gap-6'>
          <a
            href='/'
            className='font-pixel text-sm tracking-wider text-white/80 hover:text-neon'
          >
            NC
          </a>
          <nav className='flex gap-4'>
            <a href='/' className={`nav-link ${!isAdmin ? 'active' : ''}`}>
              Workspaces
            </a>
            <a href='/admin' className={`nav-link ${isAdmin ? 'active' : ''}`}>
              Admin
            </a>
          </nav>
        </div>
        <div className='flex items-center gap-3'>
          {user && <span className='text-xs text-white/40'>{user.email}</span>}
          <button onClick={handleLogout} className='btn btn-ghost text-xs'>
            Logout
          </button>
        </div>
      </div>
    </header>
  )
}

export default Dashboard
