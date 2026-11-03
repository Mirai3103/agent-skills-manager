import { useEffect, useState, type FormEvent } from 'react'
import { Events, WML } from '@wailsio/runtime'
import { GreetService } from '../bindings/github.com/Mirai3103/agent-skills-manager'

const wailsVersion = 'v3.0.0'

export default function App() {
  const [name, setName] = useState('')
  const [time, setTime] = useState('Listening for Time event...')
  const [result, setResult] = useState('')

  useEffect(() => {
    Events.On('time', (timeValue) => setTime(timeValue.data))
    WML.Reload()
  }, [])

  const doGreet = (e: FormEvent) => {
    e.preventDefault()
    GreetService.Greet(name || 'anonymous').then(setResult).catch(console.error)
  }

  return (
    <main className="flex min-h-dvh flex-col items-center justify-center gap-6 p-10">
      <h1 className="text-5xl font-extrabold tracking-tight">
        Wails +{' '}
        <span className="bg-gradient-to-r from-red-500 to-pink-500 bg-clip-text text-transparent">
          {name || 'React'}
        </span>
      </h1>
      <p className="max-w-md text-center text-neutral-400">
        Build beautiful cross-platform apps with Go and React.
      </p>

      <form
        onSubmit={doGreet}
        className="flex w-full max-w-sm items-center gap-2 rounded-xl border border-white/15 bg-white/5 py-1 pr-1 pl-3 backdrop-blur [--wails-draggable:no-drag] focus-within:border-red-500/60"
      >
        <input
          aria-label="input"
          className="min-w-0 flex-1 bg-transparent text-base outline-none select-text placeholder:text-neutral-500"
          placeholder="Your name"
          autoComplete="off"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <button
          type="submit"
          aria-label="greet-btn"
          className="rounded-lg bg-gradient-to-r from-red-500 to-pink-500 px-4 py-1.5 text-sm font-semibold text-white transition active:scale-95"
        >
          Greet
        </button>
      </form>

      <footer className="fixed inset-x-0 bottom-0 flex items-center justify-between px-3 py-3 text-xs text-neutral-400">
        <span>{wailsVersion}</span>
        <span>{time}</span>
        <a
          href="https://v3.wails.io"
          target="_blank"
          rel="noreferrer"
          className="transition hover:text-neutral-100 [--wails-draggable:no-drag]"
        >
          Docs
        </a>
      </footer>

      <div
        role="status"
        aria-live="polite"
        className={`fixed top-4 right-4 max-w-xs rounded-xl border border-white/15 bg-neutral-900/80 px-4 py-2.5 text-sm backdrop-blur transition ${
          result ? 'translate-y-0 opacity-100' : 'pointer-events-none -translate-y-4 opacity-0'
        }`}
      >
        <span className="block text-[10px] font-bold tracking-widest text-neutral-400 uppercase">From Go</span>
        <span aria-label="result" className="select-text">{result}</span>
      </div>
    </main>
  )
}
