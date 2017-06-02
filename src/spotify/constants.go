package spotify

const (
	SPOTIFY_CLIENT_ID     = "d84f9faa18a84162ad6c73697990386c"
	SPOTIFY_CLIENT_SECRET = "8f40647775b8401a866e69e3f0044bf7"
	SPOTIFY_REDIRECT_URI  = "http://localhost:8080/callback"
	SPOTIFY_FAVICON_URL   = "https://github.com/wedeploy/demo-spotify/raw/master/public/favicon.ico"
	SPOTIFY_HTML_TEMPLATE = "<!DOCTYPE html><html><head><meta http-equiv=\"Content-Type\" content=\"text/html; charset=windows-1252\"><title>Spotify DL</title><link rel=\"icon\" href=\"" + SPOTIFY_FAVICON_URL + "\" type=\"image/x-icon\" /><style>body { text-align: center; padding: 150px; } h1 { font-size: 50px; } body { font: 20px Helvetica, sans-serif; color: #333; } article { display: block; text-align: left; width: 650px; margin: 0 auto; } a { color: #dc8100; text-decoration: none; } a:hover { color: #333; text-decoration: none; }</style></head><body><article><h1>%s</h1><div><p>%s</p><br><br><p>The team.</p></div></article></body></html>"
)
