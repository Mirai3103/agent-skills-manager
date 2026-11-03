---
tags: [skill-creator]
max_turns: 60
timeout_seconds: 1800
allowed_tools: [Read, Glob, Grep, Skill, Write, Edit, Bash]
---

Trong project Wails v3 này (React + TS, thư mục notesapp/), thêm một NotesService: List/Get/Save/Delete note (id, title, body, updatedAt), lưu xuống file JSON trong thư mục config của user. Thêm nút Export để mở save dialog native và ghi note đang chọn ra file .md. Khi đóng cửa sổ mà còn thay đổi chưa lưu thì hỏi xác nhận bằng dialog native. Cập nhật frontend gọi qua bindings, generate bindings và đảm bảo go build + tsc đều pass.

Input files: notesapp/
