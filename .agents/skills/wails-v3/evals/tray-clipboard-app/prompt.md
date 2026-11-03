---
tags: [skill-creator]
max_turns: 60
timeout_seconds: 1800
allowed_tools: [Read, Glob, Grep, Skill, Write, Edit, Bash]
---

Tạo giúp mình một app desktop bằng Wails v3 tên `clipnest` (module github.com/laffy/clipnest) trong thư mục hiện tại. App chạy ở system tray, lúc khởi động không hiện cửa sổ. Có global shortcut Ctrl+Shift+V để bật/tắt một cửa sổ nhỏ hiển thị lịch sử clipboard (text) gần nhất; bên Go poll clipboard mỗi giây và đẩy item mới lên UI qua event. Đóng cửa sổ thì chỉ ẩn chứ không thoát app; menu tray có Show và Quit. Frontend vanilla TypeScript là được. Làm sao cho build pass nhé.
