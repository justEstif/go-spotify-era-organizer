package lastfm

// Tag represents a Last.fm tag with popularity count.
type Tag struct {
	Name  string `json:"name"`
	Count int    `json:"count,omitempty"` // Present in track.getTopTags, absent in artist.getTopTags
	URL   string `json:"url"`
}

// trackTagsResponse is the JSON response for track.getTopTags.
type trackTagsResponse struct {
	TopTags struct {
		Tag  []Tag `json:"tag"`
		Attr struct {
			Artist string `json:"artist"`
			Track  string `json:"track"`
		} `json:"@attr"`
	} `json:"toptags"`
}

// artistTagsResponse is the JSON response for artist.getTopTags.
type artistTagsResponse struct {
	TopTags struct {
		Tag  []Tag `json:"tag"`
		Attr struct {
			Artist string `json:"artist"`
		} `json:"@attr"`
	} `json:"toptags"`
}

// apiError represents a Last.fm API error response.
type apiError struct {
	Error   int    `json:"error"`
	Message string `json:"message"`
}
