Output media/view-secret.gif
Set Theme "Catppuccin Mocha"
Set CursorBlink false
Set Width 1400

Hide
Type "# Secrets in current namespace"
Enter 2
Show
Sleep 2
Type "kubectl get secret"
Enter
Sleep 2
Type "clear"
Enter

Hide
Type "# Showing all secrets in current namespace with interactive selection"
Enter 2
Show
Sleep 2
Type "kubectl view-secret"
Enter
Sleep 2
Down
Sleep 1
Enter
Sleep 1
Type "clear"
Enter

Hide
Type "# View 'test' secret in current namespace with interactive selection for a single key"
Enter 2
Show
Sleep 2
Type "kubectl view-secret test"
Enter
Sleep 1
Type "/key2"
Sleep 1
Enter
Sleep 1
Type "clear"
Enter

Hide
Type "# View 'test' secret in current namespace and show all keys"
Enter 2
Show
Sleep 2
Type "kubectl view-secret test -a"
Enter
Sleep 1
Type "clear"
Enter

Hide
Type "# View 'test2' secret in current namespace immediately shows the secret contents (only one key exists)"
Enter 2
Show
Sleep 2
Type "kubectl view-secret test2"
Enter
Sleep 1
Type "clear"
Enter

Hide
Type "# Secrets in 'another' namespace"
Enter 2
Show
Sleep 2
Type "kubectl get secret -n another"
Enter
Sleep 2
Type "clear"
Enter

Hide
Type "# View 'gopher' secret in 'another' namespace and silence info about single key"
Enter 2
Show
Sleep 2
Type "kubectl view-secret gopher -n another -q"
Enter

Sleep 5
