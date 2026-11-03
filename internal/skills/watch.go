package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchDirs lists every directory whose change can alter the inventory: each
// existing skills root and the Store, plus the skill directories inside them,
// and the app directory so a registered root override gets picked up. Watching
// a skill directory is what catches an edited SKILL.md.
func WatchDirs(home string, adapters []Adapter) []string {
	extras, _ := ExtraRoots(home)

	var dirs []string
	walked := make(map[string]int)
	for _, spec := range rootSpecs(home, adapters, extras) {
		dirs = appendWatched(dirs, walked, spec.Path, spec.depth)
	}
	dirs = appendWatched(dirs, walked, StoreDir(home), 1)
	return appendWatched(dirs, walked, appDir(home), 0)
}

// appendWatched appends dir and the directories below it, down to depth.
// walked remembers how deep each directory was descended so far: a depth-2 root
// nested under a depth-1 root is walked again, not skipped as already seen.
func appendWatched(dirs []string, walked map[string]int, dir string, depth int) []string {
	if seen, ok := walked[dir]; ok && seen >= depth {
		return dirs
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return dirs
	}
	if _, ok := walked[dir]; !ok {
		dirs = append(dirs, dir)
	}
	walked[dir] = depth
	if depth < 1 {
		return dirs
	}
	children, err := os.ReadDir(dir)
	if err != nil {
		return dirs
	}
	for _, child := range children {
		if child.IsDir() {
			dirs = appendWatched(dirs, walked, filepath.Join(dir, child.Name()), depth-1)
		}
	}
	return dirs
}

// Watcher turns filesystem changes under a set of directories into one refresh
// per burst. It never owns the truth: a missed event only leaves the matrix
// stale until the next refresh or Rescan.
type Watcher struct {
	fs      *fsnotify.Watcher
	quiet   time.Duration
	refresh func() []string
	report  func(error)
	watched map[string]bool
	resync  chan []string
	stop    chan struct{}
	stopped chan struct{}
}

// NewWatcher watches dirs. quiet is how long a burst must stay silent before
// refresh runs; refresh returns the directories to watch from then on, so a
// skill or root that appeared since the last pass gets watched too. report is
// given every failure, once from NewWatcher and then from the watch goroutine;
// watching continues wherever it still can.
func NewWatcher(dirs []string, quiet time.Duration, refresh func() []string, report func(error)) (*Watcher, error) {
	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{
		fs:      fs,
		quiet:   quiet,
		refresh: refresh,
		report:  report,
		watched: make(map[string]bool),
		resync:  make(chan []string),
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	w.sync(dirs)
	go w.run()
	return w, nil
}

// Sync moves the watched set to exactly dirs without refreshing. The service
// uses it after a write that changes which roots exist, so their changes are
// seen at once instead of after an unrelated event.
func (w *Watcher) Sync(dirs []string) {
	select {
	case w.resync <- dirs:
	case <-w.stopped:
	}
}

// Close stops watching and waits for the watch goroutine to exit.
func (w *Watcher) Close() {
	close(w.stop)
	w.fs.Close()
	<-w.stopped
}

// sync moves the watched set to exactly dirs.
func (w *Watcher) sync(dirs []string) {
	keep := make(map[string]bool, len(dirs))
	for _, dir := range dirs {
		keep[dir] = true
		if w.watched[dir] {
			continue
		}
		if err := w.fs.Add(dir); err != nil {
			// A directory removed between listing and adding needs no watch,
			// and must not be reported as a broken one.
			if _, statErr := os.Stat(dir); statErr != nil {
				continue
			}
			w.report(fmt.Errorf("watch %s: %w", dir, err))
			continue
		}
		w.watched[dir] = true
	}
	for dir := range w.watched {
		if !keep[dir] {
			// A directory that is gone needs no unnoticed removal.
			_ = w.fs.Remove(dir)
			delete(w.watched, dir)
		}
	}
}

func (w *Watcher) run() {
	defer close(w.stopped)

	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	var burst <-chan time.Time

	for {
		select {
		case <-w.stop:
			return
		case _, ok := <-w.fs.Events:
			if !ok {
				return
			}
			timer.Reset(w.quiet)
			burst = timer.C
		case err, ok := <-w.fs.Errors:
			if !ok {
				return
			}
			w.report(err)
		case dirs := <-w.resync:
			w.sync(dirs)
		case <-burst:
			burst = nil
			w.sync(w.refresh())
		}
	}
}
