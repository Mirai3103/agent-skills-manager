import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { AlertTriangle, ArchiveRestore, Check, Link2Off, Lock, RefreshCw } from 'lucide-react'
import { InventoryService } from '../bindings/github.com/Mirai3103/agent-skills-manager'
import {
  State,
  Status,
  type Inventory,
  type Root,
  type Skill,
} from '../bindings/github.com/Mirai3103/agent-skills-manager/internal/skills/models'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { cn } from '@/lib/utils'

export default function App() {
  const [inventory, setInventory] = useState<Inventory | null>(null)
  const [loading, setLoading] = useState(true)
  const [scanError, setScanError] = useState<string | null>(null)
  const [adopting, setAdopting] = useState<string | null>(null)
  const [adoptError, setAdoptError] = useState<string | null>(null)

  const scan = useCallback(async () => {
    setLoading(true)
    try {
      setInventory(await InventoryService.Scan())
      setScanError(null)
    } catch (err) {
      setScanError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }, [])

  const adopt = useCallback(
    async (name: string) => {
      setAdopting(name)
      setAdoptError(null)
      try {
        await InventoryService.Adopt(name)
      } catch (err) {
        setAdoptError(`${name}: ${err instanceof Error ? err.message : String(err)}`)
      } finally {
        setAdopting(null)
        // Rescan either way: a reported failure may still have changed disk.
        await scan()
      }
    },
    [scan],
  )

  useEffect(() => {
    void scan()
  }, [scan])

  const roots = inventory?.roots ?? []
  const skills = inventory?.skills ?? []
  const rootErrors = inventory?.errors ?? []

  return (
    <main className="flex min-h-dvh flex-col bg-background text-foreground">
      <header className="flex items-center justify-between gap-4 border-b px-6 py-4 [--wails-draggable:drag]">
        <div>
          <h1 className="text-base font-semibold tracking-tight">Skills matrix</h1>
          <p className="text-xs text-muted-foreground">
            {skills.length} skills · {roots.length} skills roots
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => void scan()}
          disabled={loading}
          aria-busy={loading}
          className="[--wails-draggable:no-drag]"
        >
          <RefreshCw className={cn(loading && 'animate-spin')} />
          {loading ? 'Scanning…' : 'Rescan'}
        </Button>
      </header>

      <div className="flex flex-1 flex-col gap-3 px-6 py-4">
        {scanError && (
          <Banner tone="error" title="Scan failed">
            {scanError}
          </Banner>
        )}
        {adoptError && (
          <Banner tone="error" title="Adopt failed">
            {adoptError}
          </Banner>
        )}
        {rootErrors.length > 0 && (
          <Banner tone="warning" title="Some roots could not be read">
            <ul className="list-inside list-disc">
              {rootErrors.map((rootError) => (
                <li key={rootError.root}>
                  <span className="font-mono">{rootError.root}</span> — {rootError.err}
                </li>
              ))}
            </ul>
          </Banner>
        )}

        {!inventory ? (
          <p className="py-12 text-center text-sm text-muted-foreground" role="status">
            {loading ? 'Scanning skills roots…' : 'Nothing scanned yet.'}
          </p>
        ) : roots.length === 0 && skills.length === 0 ? (
          <p className="py-12 text-center text-sm text-muted-foreground" role="status">
            No skills roots found under your home directory.
          </p>
        ) : (
          <div className="overflow-x-auto border">
            <Table aria-busy={loading} className={cn(loading && 'opacity-60')}>
              <TableHeader>
                <TableRow>
                  <TableHead scope="col" className="min-w-44">
                    Skill
                  </TableHead>
                  {roots.map((root) => (
                    <TableHead key={root.path} scope="col" className="min-w-32 align-top">
                      <span className="font-mono text-xs">{root.label}</span>
                      <span className="mt-0.5 block whitespace-normal text-[10px] leading-tight font-normal text-muted-foreground">
                        {(root.readers ?? []).join(', ')}
                      </span>
                      {!root.linkable && (
                        <Badge variant="outline" className="mt-1 gap-1 text-[10px]">
                          <Lock aria-hidden="true" />
                          read-only
                        </Badge>
                      )}
                    </TableHead>
                  ))}
                  <TableHead scope="col" className="min-w-24">
                    Status
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {skills.map((skill) => (
                  <TableRow key={skill.name}>
                    <TableHead scope="row" className="align-top whitespace-normal">
                      <span className="text-xs font-medium">{skill.name}</span>
                      {skill.hash && (
                        <span className="block font-mono text-[10px] font-normal text-muted-foreground">
                          {skill.hash.slice(0, 8)}
                        </span>
                      )}
                      {skill.warning && (
                        <span className="mt-1 flex items-start gap-1 text-[10px] leading-tight font-normal text-destructive">
                          <AlertTriangle aria-hidden="true" className="mt-px size-3 shrink-0" />
                          {skill.warning}
                        </span>
                      )}
                      {skill.stored && (
                        <Badge variant="outline" className="mt-1 text-[10px]">
                          in store
                        </Badge>
                      )}
                      <AdoptControl
                        skill={skill}
                        roots={roots}
                        adopting={adopting}
                        onAdopt={adopt}
                      />
                    </TableHead>
                    {roots.map((root, index) => (
                      <TableCell
                        key={root.path}
                        className={cn(
                          'text-center',
                          !root.linkable && 'bg-muted/50',
                        )}
                      >
                        <StateGlyph cell={skill.cells?.[index]} root={root} skill={skill} />
                      </TableCell>
                    ))}
                    <TableCell>
                      <StatusBadge status={skill.status} />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            {skills.length === 0 && (
              <p className="py-8 text-center text-sm text-muted-foreground" role="status">
                No skills found in the detected roots.
              </p>
            )}
          </div>
        )}
      </div>
    </main>
  )
}

function AdoptControl({
  skill,
  roots,
  adopting,
  onAdopt,
}: {
  skill: Skill
  roots: Root[]
  adopting: string | null
  onAdopt: (name: string) => void
}) {
  // Offered only where a real copy sits in a root. Store-only rows have none.
  const loose = (skill.cells ?? []).some((cell) => cell.state === State.Loose)
  if (!loose) return null

  const blocker = adoptBlocker(skill, roots)
  return (
    <Button
      variant="outline"
      size="sm"
      className="mt-2"
      disabled={blocker !== null || adopting !== null}
      title={blocker ?? undefined}
      onClick={() => onAdopt(skill.name)}
    >
      <ArchiveRestore aria-hidden="true" />
      {adopting === skill.name ? 'Adopting…' : 'Adopt'}
      <span className="sr-only"> {skill.name}</span>
    </Button>
  )
}

// adoptBlocker disables the obvious refusals; the backend remains the authority.
function adoptBlocker(skill: Skill, roots: Root[]): string | null {
  if (skill.status === Status.StatusVariant) {
    return 'Pick the canonical copy before adopting'
  }
  const inReadOnlyRoot = (skill.cells ?? []).some(
    (cell, index) => cell.state === State.Loose && !roots[index]?.linkable,
  )
  return inReadOnlyRoot ? 'A loose copy sits in a read-only root' : null
}

function StateGlyph({
  cell,
  root,
  skill,
}: {
  cell: { state: State; path: string } | undefined
  root: Root
  skill: Skill
}) {
  const state = cell?.state ?? State.Absent
  const name = `${skill.name} at ${root.label}: ${stateLabel(state)}${root.linkable ? '' : ', read-only root'}`

  return (
    <span
      className="inline-flex size-5 items-center justify-center"
      title={cell?.path || undefined}
      data-state={state}
    >
      <span aria-hidden="true">
        {state === State.Enabled && (
          <span className="grid size-4 place-items-center bg-primary text-primary-foreground">
            <Check className="size-3" strokeWidth={3} />
          </span>
        )}
        {state === State.Loose && (
          <span className="block size-2.5 rotate-45 border border-foreground/70" />
        )}
        {state === State.Absent && <span className="block h-px w-3 bg-border" />}
        {state === State.Orphan && (
          <span className="grid size-4 place-items-center border border-dashed border-destructive text-destructive">
            <Link2Off className="size-2.5" />
          </span>
        )}
      </span>
      <span className="sr-only">{name}</span>
    </span>
  )
}

function StatusBadge({ status }: { status: Status }) {
  switch (status) {
    case Status.StatusVariant:
      return <Badge variant="destructive">variant</Badge>
    case Status.StatusDuplicate:
      return <Badge variant="secondary">duplicate</Badge>
    case Status.StatusOrphan:
      return <Badge variant="destructive">orphan link</Badge>
    default:
      return <span className="text-muted-foreground">—</span>
  }
}

function stateLabel(state: State): string {
  switch (state) {
    case State.Enabled:
      return 'enabled'
    case State.Loose:
      return 'loose'
    case State.Orphan:
      return 'orphan link'
    default:
      return 'absent'
  }
}

function Banner({
  tone,
  title,
  children,
}: {
  tone: 'warning' | 'error'
  title: string
  children: ReactNode
}) {
  return (
    <div
      role="alert"
      className={cn(
        'flex gap-2 border px-3 py-2 text-xs',
        tone === 'error'
          ? 'border-destructive/40 bg-destructive/10 text-destructive'
          : 'border-border bg-muted/60',
      )}
    >
      <AlertTriangle aria-hidden="true" className="mt-px size-3.5 shrink-0" />
      <div className="space-y-1">
        <p className="font-medium">{title}</p>
        <div className="text-muted-foreground">{children}</div>
      </div>
    </div>
  )
}
