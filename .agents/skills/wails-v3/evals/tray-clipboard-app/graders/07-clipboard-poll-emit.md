---
type: llm
focus: trace
weight: 1
---

Go polls app.Clipboard.Text() about once per second and pushes new items with app.Event.Emit (or window EmitEvent).
