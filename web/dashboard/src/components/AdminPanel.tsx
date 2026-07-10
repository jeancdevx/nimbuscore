import { useCallback, useEffect, useState } from 'react'

import type { Team, User } from '../types'
import { api } from './api'

function AdminPanel() {
  const [users, setUsers] = useState<User[]>([])
  const [teams, setTeams] = useState<Team[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const [u, t] = await Promise.all([api.users.list(), api.teams.list()])
      setUsers(u)
      setTeams(t)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  return (
    <div className='min-h-screen'>
      <NavBar />
      <main className='mx-auto w-full max-w-6xl px-4 pb-16'>
        <header className='pt-8 pb-6 text-center'>
          <p className='font-pixel text-[10px] tracking-[0.3em] text-neon sm:text-xs'>
            ADMIN PANEL
          </p>
          <h1 className='font-pixel text-3xl leading-tight sm:text-5xl lg:text-6xl'>
            NIMBUS<span className='text-neon'>CORE</span>
          </h1>
        </header>

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

        {loading ? (
          <div className='flex justify-center py-16'>
            <div className='h-8 w-8 animate-spin rounded-full border-4 border-white/10 border-t-neon' />
          </div>
        ) : (
          <div className='grid gap-8 lg:grid-cols-2'>
            <section className='rounded-xl border border-white/5 bg-pitch-900/30 p-4'>
              <h2 className='font-pixel mb-4 text-sm tracking-wider text-white/60'>
                USERS
              </h2>
              <div className='overflow-hidden rounded-lg border border-white/5'>
                <table className='data-table'>
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Email</th>
                      <th>Provider</th>
                    </tr>
                  </thead>
                  <tbody>
                    {users.map(u => (
                      <tr key={u.id}>
                        <td className='font-medium text-white/80'>{u.name}</td>
                        <td className='text-white/60'>{u.email}</td>
                        <td className='text-white/40'>{u.provider}</td>
                      </tr>
                    ))}
                    {users.length === 0 && (
                      <tr>
                        <td
                          colSpan={3}
                          className='py-8 text-center text-white/30'
                        >
                          No users
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </section>

            <section className='rounded-xl border border-white/5 bg-pitch-900/30 p-4'>
              <h2 className='font-pixel mb-4 text-sm tracking-wider text-white/60'>
                TEAMS
              </h2>
              <div className='overflow-hidden rounded-lg border border-white/5'>
                <table className='data-table'>
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Slug</th>
                    </tr>
                  </thead>
                  <tbody>
                    {teams.map(t => (
                      <tr key={t.id}>
                        <td className='font-medium text-white/80'>{t.name}</td>
                        <td className='text-white/60'>{t.slug}</td>
                      </tr>
                    ))}
                    {teams.length === 0 && (
                      <tr>
                        <td
                          colSpan={2}
                          className='py-8 text-center text-white/30'
                        >
                          No teams
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </section>
          </div>
        )}
      </main>
    </div>
  )
}

function NavBar() {
  const handleLogout = () => {
    api.clearAuth()
    window.location.href = '/login'
  }

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
            <a href='/' className='nav-link'>
              Workspaces
            </a>
            <a href='/admin' className='nav-link active'>
              Admin
            </a>
          </nav>
        </div>
        <button onClick={handleLogout} className='btn btn-ghost text-xs'>
          Logout
        </button>
      </div>
    </header>
  )
}

export default AdminPanel
