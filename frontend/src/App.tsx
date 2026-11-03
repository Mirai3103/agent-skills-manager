import { useCallback, useEffect, useState, type FormEvent, type ReactNode } from 'react'
import {
  AlertTriangle,
  ArchiveRestore,
  Check,
  Link2Off,
  Lock,
  Plus,
  RefreshCw,
  Trash2,
} from 'lucide-react'
import { Events } from '@wailsio/runtime'
import { InventoryService } from '../bindings/github.com/Mirai3103/agent-skills-manager'
import {
  State,
  Status,
  type Cell,
  type Inventory,
  type Root,
  type Skill,
} from '../bindings/github.com/Mirai3103/agent-skills-manager/internal/skills/models'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
  const [linking, setLinking] = useState<string | null>(null)
  const [linkError, setLinkError] = useState<string | null>(null)
  const [pendingLink, setPendingLink] = useState<PendingLink | null>(null)
  const [newRoot, setNewRoot] = useState('')
  const [rootBusy, setRootBusy] = useState(false)
  const [rootError, setRootError] = useState<string | null>(null)

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

  // runAdopt runs one adopt-shaped write, then rescans whatever happened.
  const runAdopt = useCallback(
    async (name: string, action: () => Promise<unknown>) => {
      setAdopting(name)
      setAdoptError(null)
      try {
        await action()
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

  // runLink performs one enable/disable. An unconfirmed call that returns
  // warnings stops at the banner; Proceed repeats it with confirm=true.
  const runLink = useCallback(
    async (action: LinkAction, name: string, root: Root, confirm = false) => {
      setLinking(linkKey(action, name, root))
      setLinkError(null)
      setPendingLink(null)
      try {
        const result =
          action === 'enable'
            ? await InventoryService.Enable(name, root.path, confirm)
            : await InventoryService.Disable(name, root.path, confirm)
        const warnings = result.warnings ?? []
        if (warnings.length > 0 && !confirm) {
          setPendingLink({ action, skill: name, root, warnings })
          return
        }
        setPendingLink(null)
        await scan()
      } catch (err) {
        setLinkError(
          `${action === 'enable' ? 'Enable' : 'Disable'} ${name} at ${root.label}: ${
            err instanceof Error ? err.message : String(err)
          }`,
        )
        // A refused write changes nothing, but a rescan keeps the matrix honest.
        await scan()
      } finally {
        setLinking(null)
      }
    },
    [scan],
  )

  // addRoot registers one root override, then rescans so its column appears.
  const addRoot = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      const path = newRoot.trim()
      if (path === '') return
      setRootBusy(true)
      setRootError(null)
      try {
        await InventoryService.AddExtraRoot(path)
        setNewRoot('')
        await scan()
      } catch (err) {
        setRootError(`Add ${path}: ${err instanceof Error ? err.message : String(err)}`)
      } finally {
        setRootBusy(false)
      }
    },
    [newRoot, scan],
  )

  // removeRoot forgets the override; the skills under it stay on disk.
  const removeRoot = useCallback(
    async (path: string) => {
      setRootBusy(true)
      setRootError(null)
      try {
        await InventoryService.RemoveExtraRoot(path)
        await scan()
      } catch (err) {
        setRootError(`Remove ${path}: ${err instanceof Error ? err.message : String(err)}`)
      } finally {
        setRootBusy(false)
      }
    },
    [scan],
  )

  useEffect(() => {
    void scan()
  }, [scan])

  // Live refresh: the backend watches the roots and the store, collapses a
  // burst into one event, and sends the freshly scanned inventory. A missed
  // event only leaves the matrix stale; Rescan still works.
  useEffect(() => {
    return Events.On('inventory:changed', (event) => {
      setInventory(event.data as Inventory)
      setScanError(null)
    })
  }, [])

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
        <div className="flex items-center gap-2 [--wails-draggable:no-drag]">
          <form className="flex items-center gap-2" onSubmit={(event) => void addRoot(event)}>
            <Input
              value={newRoot}
              onChange={(event) => setNewRoot(event.target.value)}
              placeholder="/path/to/skills or ~/skills"
              aria-label="Skills root to add"
              className="w-64"
              spellCheck={false}
            />
            <Button
              type="submit"
              variant="outline"
              size="sm"
              disabled={rootBusy || newRoot.trim() === ''}
            >
              <Plus aria-hidden="true" />
              Add root
            </Button>
          </form>
          <Button
            variant="outline"
            size="sm"
            onClick={() => void scan()}
            disabled={loading}
            aria-busy={loading}
          >
            <RefreshCw className={cn(loading && 'animate-spin')} />
            {loading ? 'Scanning…' : 'Rescan'}
          </Button>
        </div>
      </header>

      <div className="flex flex-1 flex-col gap-3 px-6 py-4">
        {scanError && (
          <Banner tone="error" title="Scan failed">
            {scanError}
          </Banner>
        )}
        {inventory?.watchError && (
          <Banner tone="error" title="Live refresh unavailable">
            {inventory.watchError} — Rescan is how the matrix stays current.
          </Banner>
        )}
        {adoptError && (
          <Banner tone="error" title="Adopt failed">
            {adoptError}
          </Banner>
        )}
        {linkError && (
          <Banner tone="error" title="Link change failed">
            {linkError}
          </Banner>
        )}
        {rootError && (
          <Banner tone="error" title="Extra root change failed">
            {rootError}
          </Banner>
        )}
        {pendingLink && (
          <Banner
            tone="warning"
            title={`${pendingLink.action === 'enable' ? 'Enable' : 'Disable'} ${
              pendingLink.skill
            } at ${pendingLink.root.label}?`}
          >
            <ul className="list-inside list-disc">
              {pendingLink.warnings.map((warning) => (
                <li key={warning}>{warning}</li>
              ))}
            </ul>
            <div className="mt-2 flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={linking !== null}
                onClick={() =>
                  void runLink(pendingLink.action, pendingLink.skill, pendingLink.root, true)
                }
              >
                Proceed
                <span className="sr-only">
                  {' '}
                  with {pendingLink.action} {pendingLink.skill} at {pendingLink.root.label}
                </span>
              </Button>
              <Button variant="ghost" size="sm" onClick={() => setPendingLink(null)}>
                Cancel
              </Button>
            </div>
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
                        {(root.readers ?? []).length > 0
                          ? (root.readers ?? []).join(', ')
                          : 'no adapter claims this root'}
                      </span>
                      {root.extra && (
                        <span className="mt-1 flex items-center gap-1">
                          <Badge variant="outline" className="text-[10px]">
                            extra root
                          </Badge>
                          <Button
                            variant="ghost"
                            size="icon-xs"
                            aria-label={`Remove extra root ${root.label}`}
                            title="Forget this root. Nothing on disk changes."
                            disabled={rootBusy}
                            onClick={() => void removeRoot(root.path)}
                          >
                            <Trash2 aria-hidden="true" />
                          </Button>
                        </span>
                      )}
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
                        onAdopt={(name) => void runAdopt(name, () => InventoryService.Adopt(name))}
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
                        <StateGlyph
                          cell={skill.cells?.[index]}
                          root={root}
                          skill={skill}
                          adopting={adopting}
                          linking={linking}
                          onLink={runLink}
                          onChoose={(name, path) =>
                            void runAdopt(name, () => InventoryService.ChooseCanonical(name, path))
                          }
                        />
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
    return 'Pick the canonical copy in its column'
  }
  if ((skill.cells ?? []).some((cell) => cell.conflict)) {
    return 'Remove the conflicting copy by hand, then rescan'
  }
  const inReadOnlyRoot = (skill.cells ?? []).some(
    (cell, index) => cell.state === State.Loose && !roots[index]?.linkable,
  )
  return inReadOnlyRoot ? 'A loose copy sits in a read-only root' : null
}

type LinkAction = 'enable' | 'disable'

// linkKey identifies the one in-flight write, so only its cell shows busy.
function linkKey(action: LinkAction, name: string, root: Root): string {
  return `${action}:${name}:${root.path}`
}

type PendingLink = {
  action: LinkAction
  skill: string
  root: Root
  warnings: string[]
}

function StateGlyph({
  cell,
  root,
  skill,
  adopting,
  linking,
  onLink,
  onChoose,
}: {
  cell: Cell | undefined
  root: Root
  skill: Skill
  adopting: string | null
  linking: string | null
  onLink: (action: LinkAction, name: string, root: Root) => void
  onChoose: (name: string, path: string) => void
}) {
  const state = cell?.state ?? State.Absent
  const conflict = cell?.conflict === true
  const status = conflict
    ? `${skill.name} at ${root.label}: conflicting copy at ${cell?.path}`
    : `${skill.name} at ${root.label}: ${stateLabel(state)}${root.linkable ? '' : ', read-only root'}`
  const chooseable = skill.status === Status.StatusVariant && state === State.Loose && root.linkable && cell?.path
  // The cell is the control (ADR-0004): absent enables, linked disables. A
  // read-only, conflicting, or not-yet-adopted cell has no write affordance.
  const action: LinkAction | null =
    conflict || !root.linkable
      ? null
      : state === State.Absent && skill.stored
        ? 'enable'
        : state === State.Enabled
          ? 'disable'
          : null

  const mark = conflict ? (
    <span className="grid size-4 place-items-center border border-destructive text-destructive">
      <AlertTriangle className="size-2.5" />
    </span>
  ) : (
    <>
      {state === State.Enabled && (
        <span className="grid size-4 place-items-center bg-primary text-primary-foreground">
          <Check className="size-3" strokeWidth={3} />
        </span>
      )}
      {state === State.Loose && <span className="block size-2.5 rotate-45 border border-foreground/70" />}
      {state === State.Absent && <span className="block h-px w-3 bg-border" />}
      {state === State.Orphan && (
        <span className="grid size-4 place-items-center border border-dashed border-destructive text-destructive">
          <Link2Off className="size-2.5" />
        </span>
      )}
    </>
  )

  return (
    <span className="inline-flex flex-col items-center">
      {action ? (
        <Button
          variant="ghost"
          size="icon-xs"
          title={cell?.path || undefined}
          aria-label={`${action === 'enable' ? 'Enable' : 'Disable'} ${skill.name} at ${root.label}`}
          aria-busy={linking === linkKey(action, skill.name, root)}
          disabled={linking !== null}
          data-state={state}
          onClick={() => onLink(action, skill.name, root)}
        >
          <span aria-hidden="true">{mark}</span>
        </Button>
      ) : (
        <span
          className="inline-flex size-5 items-center justify-center"
          title={cell?.path || undefined}
          data-state={state}
          data-conflict={conflict || undefined}
        >
          <span aria-hidden="true">{mark}</span>
          <span className="sr-only">{status}</span>
        </span>
      )}
      {conflict && (
        <span className="mt-1 block max-w-36 break-all font-mono text-[9px] leading-tight text-destructive">
          {cell?.path}
        </span>
      )}
      {chooseable && (
        <Button
          variant="outline"
          size="sm"
          className="mt-1 h-6 px-2 text-[10px]"
          disabled={adopting !== null || linking !== null}
          onClick={() => onChoose(skill.name, cell.path)}
        >
          Use this copy
          <span className="sr-only"> at {root.label} as canonical for {skill.name}</span>
        </Button>
      )}
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
    case Status.StatusConflict:
      return <Badge variant="destructive">conflict</Badge>
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
