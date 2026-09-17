from pathlib import Path

p = Path("internal/tui/tui.go")
s = p.read_text()

repls = [
    ("\tViewDeleteCommandConfirm\n)", "\tViewDeleteCommandConfirm\n\tViewKnownHostConfirm\n)"),
    ("\t\tcase ViewDeleteCommandConfirm:\n\t\t\treturn m.updateDeleteCommandConfirm(msg)\n\t\t}", "\t\tcase ViewDeleteCommandConfirm:\n\t\t\treturn m.updateDeleteCommandConfirm(msg)\n\t\tcase ViewKnownHostConfirm:\n\t\t\treturn m.updateKnownHostConfirm(msg)\n\t\t}"),
    ("\tcase ViewDeleteCommandConfirm:\n\t\tbody = m.viewDeleteCommandConfirm()\n\tdefault:", "\tcase ViewDeleteCommandConfirm:\n\t\tbody = m.viewDeleteCommandConfirm()\n\tcase ViewKnownHostConfirm:\n\t\tbody = m.viewKnownHostConfirm()\n\tdefault:"),
    ("type sshErrorMsg struct {\n\tTabId int\n\tMsg   string\n}\n\ntype sshConnectedMsg struct {", "type sshErrorMsg struct {\n\tTabId int\n\tMsg   string\n}\n\ntype sshHostKeyErrorMsg struct {\n\tTabId int\n\tErr   error\n}\n\ntype sshConnectedMsg struct {"),
    ("\tcase sshErrorMsg:\n\t\tm.toast = msg.Msg", "\tcase sshHostKeyErrorMsg:\n\t\tif msg.TabId == 0 {\n\t\t\tfor i := range m.tabs {\n\t\t\t\tif m.tabs[i].Connecting {\n\t\t\t\t\th := m.tabs[i].Host\n\t\t\t\t\tm.sshHost = &h\n\t\t\t\t\tm.err = msg.Err\n\t\t\t\t\tm.view = ViewKnownHostConfirm\n\t\t\t\t\treturn m, nil\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t\tm.toast = msg.Err.Error()\n\t\tm.toastSuccess = false\n\t\tm.toastTimer = 120\n\t\treturn m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })\n\n\tcase sshErrorMsg:\n\t\tm.toast = msg.Msg"),
    ("\t\tcase res := <-ch:\n\t\t\tif res.err != nil {\n\t\t\t\treturn sshErrorMsg{0, res.err.Error()}\n\t\t\t}\n\t\t\treturn sshConnectedMsg{session: res.session}", "\t\tcase res := <-ch:\n\t\t\tif res.err != nil {\n\t\t\t\tif _, ok := res.err.(*ssh.UnknownHostKeyError); ok {\n\t\t\t\t\treturn sshHostKeyErrorMsg{TabId: 0, Err: res.err}\n\t\t\t\t}\n\t\t\t\tif _, ok := res.err.(*ssh.ChangedHostKeyError); ok {\n\t\t\t\t\treturn sshHostKeyErrorMsg{TabId: 0, Err: res.err}\n\t\t\t\t}\n\t\t\t\treturn sshErrorMsg{0, res.err.Error()}\n\t\t\t}\n\t\t\treturn sshConnectedMsg{session: res.session}"),
]

for old, new in repls:
    if old not in s:
        raise SystemExit(f"expected source fragment not found: {old[:80]!r}")
    s = s.replace(old, new, 1)

p.write_text(s)
