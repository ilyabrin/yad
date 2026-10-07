# Records the README animation with VHS (https://github.com/charmbracelet/vhs).
#
# Everything on screen comes from the fake API in this folder, never from a
# real account. From the repository root:
#
#   go build -o yad . && go run ./tools/demo &
#   vhs tools/demo/demo.tape
#
# HOME points at a throwaway folder so yad does not touch your own
# ~/.yad/config.yaml when it remembers the last folder on exit.

Output docs/demo.gif

Require yad

Set Shell "bash"
Set FontSize 16
Set Width 1400
Set Height 640
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 80ms

Env YANDEX_DISK_TOKEN "demo"
Env YANDEX_DISK_API_URL "http://127.0.0.1:8765/v1/disk"

Hide
Type "export HOME=$(mktemp -d) && clear"
Enter
Show

Type "yad"
Sleep 400ms
Enter
Sleep 2s

# Into Photos, look around
Down
Sleep 300ms
Down
Sleep 300ms
Enter
Sleep 1.5s
Down@400ms 3
Sleep 600ms

# Filter by name
Type "/"
Sleep 300ms
Type "iceland"
Sleep 1s
Enter
Sleep 800ms

# Share a photo
Type "p"
Sleep 2s

# Close the link, clear the filter, go back; then disk info and the trash
Escape
Sleep 400ms
Escape
Sleep 600ms
Backspace
Sleep 1.2s
Type "i"
Sleep 2.5s
Escape
Sleep 500ms
Type "t"
Sleep 2.5s
Escape
Sleep 800ms
Type "q"
Sleep 500ms
