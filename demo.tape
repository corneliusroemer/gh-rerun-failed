# demo.tape
# Record with: vhs < demo.tape

Output demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 600
Set Padding 20
Set WindowBar Rounded
Set Theme "Charm"

# Start the demo
Sleep 1s
Type "gh rerun-failed --help"
Enter
Sleep 3s

# Clear and show a dry-run example
Clear
Type "gh rerun-failed --repo loculus-project/loculus --branch main --dry-run --limit 5"
Sleep 500ms
Enter
Sleep 10s

# Show filtering by since
Type "gh rerun-failed --repo loculus-project/loculus --since 24h --dry-run"
Sleep 500ms
Enter
Sleep 5s

# End
Sleep 2s
