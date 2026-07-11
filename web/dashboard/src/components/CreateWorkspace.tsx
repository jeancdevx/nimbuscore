import { useState } from 'react'

import type { PortMapping, Workspace } from '../types'

interface Props {
  onSubmit: (data: Partial<Workspace>) => Promise<void>
  onClose: () => void
}

function CreateWorkspace({ onSubmit, onClose }: Props) {
  const [name, setName] = useState('')
  const [image, setImage] = useState('codercom/code-server:latest')
  const [repoUrl, setRepoUrl] = useState('')
  const [branch, setBranch] = useState('main')
  const [cpu, setCpu] = useState('1')
  const [memory, setMemory] = useState('2Gi')
  const [disk, setDisk] = useState('10Gi')
  const [ports, setPorts] = useState<PortMapping[]>([])
  const [newPort, setNewPort] = useState('')
  const [newProtocol, setNewProtocol] = useState('http')
  const [newSubdomain, setNewSubdomain] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const addPort = () => {
    const port = parseInt(newPort, 10)
    if (isNaN(port) || port < 1 || port > 65535) return
    setPorts(p => [
      ...p,
      { port, protocol: newProtocol, subdomain: newSubdomain || undefined }
    ])
    setNewPort('')
    setNewSubdomain('')
  }

  const removePort = (idx: number) => {
    setPorts(p => p.filter((_, i) => i !== idx))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name) return

    setSubmitting(true)
    setError(null)
    try {
      await onSubmit({
        name,
        image: image || undefined,
        repo_url: repoUrl || undefined,
        branch: branch || undefined,
        resources: { cpu, memory, disk },
        ports: ports.length > 0 ? ports : undefined
      })
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to create workspace'
      )
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className='modal-overlay animate-fade-in' onClick={onClose}>
      <div
        className='modal-card max-h-[90vh] overflow-y-auto'
        onClick={e => e.stopPropagation()}
      >
        <div className='mb-6 flex items-center justify-between'>
          <h2 className='font-pixel text-sm tracking-widest text-neon'>
            NEW WORKSPACE
          </h2>
          <button
            type='button'
            onClick={onClose}
            className='rounded-md p-1 text-white/30 transition-colors hover:bg-white/5 hover:text-white/70'
          >
            <svg
              width='18'
              height='18'
              viewBox='0 0 24 24'
              fill='none'
              stroke='currentColor'
              strokeWidth='2'
            >
              <path d='M18 6L6 18M6 6l12 12' />
            </svg>
          </button>
        </div>

        {error && (
          <div className='mb-5 flex items-start gap-2.5 rounded-lg border border-red-500/20 bg-red-500/10 px-3.5 py-2.5 text-xs text-danger'>
            <svg
              className='mt-0.5 shrink-0'
              width='14'
              height='14'
              viewBox='0 0 24 24'
              fill='none'
              stroke='currentColor'
              strokeWidth='2'
            >
              <circle cx='12' cy='12' r='10' />
              <path d='M12 8v4M12 16h0' />
            </svg>
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className='space-y-5'>
          <div className='grid grid-cols-2 gap-3'>
            <div>
              <label className='label'>Name *</label>
              <input
                value={name}
                onChange={e => setName(e.target.value)}
                className='input'
                placeholder='my-workspace'
                required
              />
            </div>
            <div>
              <label className='label'>Image</label>
              <input
                value={image}
                onChange={e => setImage(e.target.value)}
                className='input'
                placeholder='codercom/code-server:latest'
              />
            </div>
          </div>

          <fieldset className='rounded-lg border border-white/5 bg-white/2 p-3.5'>
            <legend className='px-1 text-[10px] font-medium uppercase tracking-[0.12em] text-white/25'>
              Resources
            </legend>
            <div className='grid grid-cols-3 gap-3'>
              <div>
                <label className='label'>CPU</label>
                <input
                  value={cpu}
                  onChange={e => setCpu(e.target.value)}
                  className='input'
                  placeholder='1'
                />
                <p className='mt-0.5 text-[10px] text-white/20'>e.g. 1, 500m</p>
              </div>
              <div>
                <label className='label'>RAM</label>
                <input
                  value={memory}
                  onChange={e => setMemory(e.target.value)}
                  className='input'
                  placeholder='2Gi'
                />
                <p className='mt-0.5 text-[10px] text-white/20'>
                  e.g. 2Gi, 512Mi
                </p>
              </div>
              <div>
                <label className='label'>Disk</label>
                <input
                  value={disk}
                  onChange={e => setDisk(e.target.value)}
                  className='input'
                  placeholder='10Gi'
                />
                <p className='mt-0.5 text-[10px] text-white/20'>
                  e.g. 10Gi, 20Gi
                </p>
              </div>
            </div>
          </fieldset>

          <fieldset className='rounded-lg border border-white/5 bg-white/2 p-3.5'>
            <legend className='px-1 text-[10px] font-medium uppercase tracking-[0.12em] text-white/25'>
              Repository
            </legend>
            <div className='space-y-3'>
              <div>
                <label className='label'>Repo URL</label>
                <input
                  value={repoUrl}
                  onChange={e => setRepoUrl(e.target.value)}
                  className='input'
                  placeholder='https://github.com/user/repo'
                />
              </div>
              <div>
                <label className='label'>Branch</label>
                <input
                  value={branch}
                  onChange={e => setBranch(e.target.value)}
                  className='input'
                />
              </div>
            </div>
          </fieldset>

          <fieldset className='rounded-lg border border-white/5 bg-white/2 p-3.5'>
            <legend className='px-1 text-[10px] font-medium uppercase tracking-[0.12em] text-white/25'>
              Ports
            </legend>
            <div className='space-y-2.5'>
              {ports.length > 0 && (
                <div className='space-y-1.5'>
                  {ports.map((p, i) => (
                    <div
                      key={i}
                      className='flex items-center justify-between rounded-md border border-white/6 bg-white/3 px-3 py-2 text-xs text-white/60'
                    >
                      <div className='flex items-center gap-2'>
                        <span className='flex h-5 w-8 items-center justify-center rounded bg-neon-dim/10 text-[10px] text-neon'>
                          {p.port}
                        </span>
                        <span className='font-medium text-white/80'>
                          {p.port}/{p.protocol.toUpperCase()}
                        </span>
                        {p.subdomain && (
                          <>
                            <span className='text-white/20'>→</span>
                            <code className='rounded bg-white/5 px-1.5 py-0.5 text-[10px] text-white/50'>
                              {p.subdomain}
                            </code>
                          </>
                        )}
                      </div>
                      <button
                        type='button'
                        onClick={() => removePort(i)}
                        className='rounded p-1 text-white/20 transition-colors hover:bg-red-500/10 hover:text-danger'
                      >
                        <svg
                          width='14'
                          height='14'
                          viewBox='0 0 24 24'
                          fill='none'
                          stroke='currentColor'
                          strokeWidth='2'
                        >
                          <path d='M18 6L6 18M6 6l12 12' />
                        </svg>
                      </button>
                    </div>
                  ))}
                </div>
              )}
              <div className='flex flex-wrap items-end gap-2'>
                <div className='min-w-0 flex-1 basis-20'>
                  <label className='label'>Port</label>
                  <input
                    value={newPort}
                    onChange={e => setNewPort(e.target.value)}
                    className='input'
                    placeholder='3000'
                    type='number'
                    min='1'
                    max='65535'
                  />
                </div>

                <div className='min-w-0 basis-22.5'>
                  <label className='label'>Protocol</label>
                  <select
                    value={newProtocol}
                    onChange={e => setNewProtocol(e.target.value)}
                    className='input'
                  >
                    <option value='http'>HTTP</option>
                    <option value='https'>HTTPS</option>
                    <option value='tcp'>TCP</option>
                    <option value='udp'>UDP</option>
                  </select>
                </div>

                <div className='min-w-0 flex-1 basis-30]'>
                  <label className='label'>Subdomain</label>
                  <input
                    value={newSubdomain}
                    onChange={e => setNewSubdomain(e.target.value)}
                    className='input'
                    placeholder='optional'
                  />
                </div>

                <button
                  type='button'
                  onClick={addPort}
                  disabled={!newPort}
                  className='btn btn-neon mb-1 px-4 text-white py-2.25 text-xs disabled:opacity-80'
                >
                  Add
                </button>
              </div>
            </div>
          </fieldset>

          <div className='flex gap-3 pt-1'>
            <button
              type='button'
              onClick={onClose}
              className='btn btn-ghost flex-1 py-2.5 text-xs'
            >
              Cancel
            </button>
            <button
              type='submit'
              disabled={submitting}
              className='btn btn-neon flex-1 py-2.5 text-xs font-semibold tracking-wide disabled:opacity-50'
            >
              {submitting ? (
                <span className='flex items-center justify-center gap-2'>
                  <span className='h-3.5 w-3.5 animate-spin rounded-full border-2 border-pitch-950 border-t-transparent' />
                  Creating...
                </span>
              ) : (
                'CREATE WORKSPACE'
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default CreateWorkspace
