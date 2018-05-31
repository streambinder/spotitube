package spotify

const (
	// SpotifyClientID : Spotify app client ID
	SpotifyClientID = "d84f9faa18a84162ad6c73697990386c"
	// SpotifyClientSecret : Spotify app client secret key
	SpotifyClientSecret = ":SPOTIFY_CLIENT_SECRET:"

	// SpotifyRedirectURL : Spotify app redirect URL
	SpotifyRedirectURL = "http://localhost:8080/callback"
	// SpotifyFaviconURL : Spotify app redirect URL's favicon
	SpotifyFaviconURL = "https://raw.githubusercontent.com/streambinder/spotitube/master/assets/images/spotify.ico"
	// SpotifyHTMLAutoCloseTimeout : Spotify app redirect URL's autoclose timeout
	SpotifyHTMLAutoCloseTimeout = "5" // s
	// SpotifyHTMLAutoCloseTimeoutMs : Spotify app redirect URL's autoclose timeout in ms (automatically parsed from SpotifyHTMLAutoCloseTimeout)
	SpotifyHTMLAutoCloseTimeoutMs = SpotifyHTMLAutoCloseTimeout + "000" // ms
	// SpotifyHTMLSigAuthor : Spotify app redirect URL's footer quoted author
	SpotifyHTMLSigAuthor = "streambinder"
	// SpotifyHTMLSigIcon : Spotify app redirect URL's footer quoted author icon
	SpotifyHTMLSigIcon = "https://davidepucci.it/images/avatar.jpg"
	// SpotifyHTMLTemplate : Spotify app redirect URLS's template
	SpotifyHTMLTemplate = `<!DOCTYPE html>
	<html>
	<head>
		<meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\">
		<title>SpotiTube</title>
		<link rel="icon" href="` + SpotifyFaviconURL + `" type="image/x-icon" />
		<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css" />
		<link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Roboto+Condensed" />
		<style>
			body               { font: 20px 'Roboto Condensed', sans-serif; text-align: center; margin: 75px 0; padding: 50px; color: #333; border: solid 5px #1ED760; border-left: none; border-right: none; }
			body *             { text-align: center; }
			h1                 { font-size: 40px; text-transform: uppercase; }
			h1>i.fa            { margin: auto 10px; }
			article            { display: block; text-align: left; width: 650px; margin: 0 auto 50px; }
			a                  { color: #dc8100; text-decoration: none; }
			a:hover            { color: #333; text-decoration: none; }
			p.timer            { font-size: 14px; color: #A0A0A0; text-align: center; text-transform: uppercase; }
			div.signature      { border: 1px solid rgba(0, 0, 0, 0.05); border-radius: 5px; text-align: center; }
			div.signature>img  { width: 35px; vertical-align: middle; }
			div.signature>span { font-size: 15px; color: #505050; }
		</style>
		<script type="text/javascript">
			var timeleft = ` + SpotifyHTMLAutoCloseTimeout + `;
			var downloadTimer = setInterval(function() {
				timeleft--;
				document.getElementById("timer").textContent = timeleft;
				if(timeleft <= 0)
					clearInterval(downloadTimer);
			}, 1000);
			function setAutoClose() { window.setTimeout(autoClose, ` + SpotifyHTMLAutoCloseTimeoutMs + `); }
			function autoClose() { window.close(); }
		</script>
	</head>
	<body onLoad="setAutoClose()">
		<article>
			<h1><i class="fa fa-thumbs-up" aria-hidden="true"></i><br>%s</h1>
			<div>
				<h3>%s</h3>
				<br><br><br>
				<p class="timer">Window will attempt to close in <span id="timer">` + SpotifyHTMLAutoCloseTimeout + `</span> seconds.</p>
				<br>
				<div class="signature">
					<img src="` + SpotifyHTMLSigIcon + `"/>
					<span>` + SpotifyHTMLSigAuthor + `</span>
				</div>
			</div>
		</article>
	</body>
	</html>`
)
