package youtube

const (
	// YouTubeVideoPrefix : YouTube video prefix
	YouTubeVideoPrefix = "https://www.youtube.com"
	// YouTubeQueryURL : YouTube query URL
	YouTubeQueryURL = YouTubeVideoPrefix + "/results"
	// YouTubeQueryPattern : YouTube query URL parseable with *printf functions
	YouTubeQueryPattern = YouTubeQueryURL + "?q=%s"
	// YouTubeHTMLVideoSelector : YouTube entry video selector
	YouTubeHTMLVideoSelector = ".yt-uix-tile-link"
	// YouTubeHTMLDescSelector : YouTube entry description selector
	YouTubeHTMLDescSelector = ".yt-lockup-byline"
	// YouTubeHTMLDurationSelector : YouTube entry duration selector
	YouTubeHTMLDurationSelector = ".accessible-description"
	// YouTubeDurationTolerance : max video duration difference tolerance
	YouTubeDurationTolerance = 20 // second(s)
)
