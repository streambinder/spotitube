Output demo.gif

Require spotitube

Set Shell "bash"
Set FontSize 18
Set Width 1200
Set Height 600
Set WindowBar Colorful
Set BorderRadius 10

Hide
Type "rm -rf my-music && mkdir my-music && clear"
Enter
Show

Type "spotitube sync -o my-music --track 08Bkl20GdNhlh7UdUcee2r --playlist spotitube"
Sleep 500ms
Enter
Wait+Screen@30s /synchronization complete/
Sleep 1s

Type "spotitube show 'my-music/The Strokes - Metabolism.mp3'"
Sleep 500ms
Enter
Sleep 3s

Hide
Type "rm -rf my-music"
Enter
