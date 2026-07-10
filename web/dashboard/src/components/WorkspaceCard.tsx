import type { Workspace, WorkspaceStatus } from '../types'

const badgeClass: Record<WorkspaceStatus, string> = {
  pending: 'badge-pending',
  building: 'badge-building',
  running: 'badge-running',
  stopping: 'badge-stopping',
  stopped: 'badge-stopped',
  error: 'badge-error',
  deleted: 'badge-deleted'
}

const cardStatus: Record<WorkspaceStatus, string> = {
  pending: '',
  building: '',
  running: 'is-running',
  stopping: '',
  stopped: 'is-stopped',
  error: 'is-error',
  deleted: 'is-stopped'
}

interface Props {
  workspace: Workspace
  onStart: () => void
  onStop: () => void
  onDelete: () => void
}

function WorkspaceCard({ workspace: ws, onStart, onStop, onDelete }: Props) {
  const canStart = ws.status === 'stopped' || ws.status === 'error'
  const canStop = ws.status === 'running'
  const canDelete = ws.status !== 'deleted'

  const cpuNum = parseFloat(ws.resources.cpu) || 1
  const memNum = parseFloat(ws.resources.memory) || 2
  const diskNum = parseFloat(ws.resources.disk) || 10
  const maxCpu = 8
  const maxMem = 16
  const maxDisk = 100

  return (
    <div className={`ws-card ${cardStatus[ws.status]}`}>
      <div className='mb-3 flex items-start justify-between'>
        <div className='min-w-0 flex-1'>
          <h3 className='truncate text-sm font-medium text-white/90'>
            {ws.name}
          </h3>
          <p className='mt-0.5 text-[11px] text-white/30'>
            {new Date(ws.created_at).toLocaleDateString('en-US', {
              month: 'short',
              day: 'numeric'
            })}
          </p>
        </div>
        <span className={`badge ${badgeClass[ws.status]}`}>{ws.status}</span>
      </div>

      <div className='mb-3 space-y-2'>
        <div>
          <div className='flex items-center justify-between text-[11px]'>
            <span className='text-white/40'>CPU</span>
            <span className='tabular text-white/60'>{ws.resources.cpu}</span>
          </div>
          <div className='resource-bar'>
            <div
              className='resource-bar__fill'
              style={{ width: `${Math.min((cpuNum / maxCpu) * 100, 100)}%` }}
            />
          </div>
        </div>
        <div>
          <div className='flex items-center justify-between text-[11px]'>
            <span className='text-white/40'>RAM</span>
            <span className='tabular text-white/60'>{ws.resources.memory}</span>
          </div>
          <div className='resource-bar'>
            <div
              className='resource-bar__fill'
              style={{ width: `${Math.min((memNum / maxMem) * 100, 100)}%` }}
            />
          </div>
        </div>
        <div>
          <div className='flex items-center justify-between text-[11px]'>
            <span className='text-white/40'>DISK</span>
            <span className='tabular text-white/60'>{ws.resources.disk}</span>
          </div>
          <div className='resource-bar'>
            <div
              className='resource-bar__fill'
              style={{ width: `${Math.min((diskNum / maxDisk) * 100, 100)}%` }}
            />
          </div>
        </div>
      </div>

      {ws.repo_url && (
        <p className='mb-3 truncate text-[11px] text-white/30'>{ws.repo_url}</p>
      )}

      <div className='flex gap-2'>
        {canStart && (
          <button
            onClick={onStart}
            className='btn btn-neon flex-1 py-1.5 text-[11px]'
          >
            Start
          </button>
        )}
        {canStop && (
          <button
            onClick={onStop}
            className='btn flex-1 border border-orange-500/30 bg-orange-500/10 py-1.5 text-[11px] text-orange-400 hover:bg-orange-500/20'
          >
            Stop
          </button>
        )}
        {canDelete && (
          <button
            onClick={onDelete}
            className='btn btn-danger px-2 py-1.5 text-[11px]'
          >
            ✕
          </button>
        )}
      </div>
    </div>
  )
}

export default WorkspaceCard
