Output demo.gif

Require pango

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

Type "pango sync -o my-music --track 08Bkl20GdNhlh7UdUcee2r --playlist pango"
Sleep 500ms
Enter
Wait+Screen@30s /synchronization complete/
Sleep 1s

Type "pango show 'my-music/The Strokes - Metabolism.mp3'"
Sleep 500ms
Enter
Sleep 3s

Hide
Type "rm -rf my-music"
Enter
