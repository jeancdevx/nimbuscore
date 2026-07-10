import { useState } from 'react'

import type { Workspace } from '../types'

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
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

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
        resources: { cpu, memory, disk }
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
    <div className='modal-overlay animate-fade-in'>
      <div className='modal-card'>
        <h2 className='font-pixel mb-6 text-center text-sm tracking-widest text-neon'>
          NEW WORKSPACE
        </h2>

        {error && (
          <div className='mb-4 rounded-md border border-red-500/20 bg-red-500/10 p-3 text-xs text-danger'>
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className='space-y-4'>
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
            />
          </div>

          <div className='grid grid-cols-3 gap-3'>
            <div>
              <label className='label'>CPU</label>
              <input
                value={cpu}
                onChange={e => setCpu(e.target.value)}
                className='input'
              />
            </div>
            <div>
              <label className='label'>RAM</label>
              <input
                value={memory}
                onChange={e => setMemory(e.target.value)}
                className='input'
              />
            </div>
            <div>
              <label className='label'>Disk</label>
              <input
                value={disk}
                onChange={e => setDisk(e.target.value)}
                className='input'
              />
            </div>
          </div>

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

          <div className='flex gap-3 pt-2'>
            <button
              type='button'
              onClick={onClose}
              className='btn btn-ghost flex-1 py-2 text-xs'
            >
              Cancel
            </button>
            <button
              type='submit'
              disabled={submitting}
              className='btn btn-neon flex-1 py-2 text-xs font-semibold tracking-wide disabled:opacity-50'
            >
              {submitting ? 'Creating...' : 'CREATE'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default CreateWorkspace
