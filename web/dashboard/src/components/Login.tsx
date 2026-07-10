import { api } from './api'

function Login() {
  return (
    <div className='flex min-h-screen items-center justify-center'>
      <div className='w-full max-w-md space-y-8 px-6 text-center'>
        <div className='space-y-3'>
          <h1 className='font-pixel text-4xl leading-tight sm:text-5xl'>
            NIMBUS<span className='text-neon'>CORE</span>
          </h1>
          <p className='text-sm text-white/55'>
            Entornos de desarrollo on-demand
          </p>
        </div>

        <div className='pt-6'>
          <button
            onClick={api.login}
            className='btn btn-neon w-full px-6 py-3 text-sm font-semibold tracking-wide'
          >
            Sign in with SSO
          </button>
        </div>
      </div>
    </div>
  )
}

export default Login
