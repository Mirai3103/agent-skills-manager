---
type: llm
focus: trace
weight: 1
---

Close confirmation uses RegisterHook(events.Common.WindowClosing, ...) with e.Cancel() and a native question dialog, driven by an unsaved/dirty flag the frontend reports to Go.
