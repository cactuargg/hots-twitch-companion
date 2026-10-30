// OAuth and Helix records retain the existing persisted and remote JSON fields.
package twitch

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	SavedAt      int64  `json:"savedAt"`
}

type DeviceResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type Identity struct {
	ClientID string   `json:"client_id"`
	UserID   string   `json:"user_id"`
	Login    string   `json:"login"`
	Scopes   []string `json:"scopes"`
}

type APIPrediction struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	Title            string `json:"title"`
	WinningOutcomeID string `json:"winning_outcome_id"`
	Outcomes         []struct {
		ID            string `json:"id"`
		Title         string `json:"title"`
		ChannelPoints *int64 `json:"channel_points"`
	} `json:"outcomes"`
}

type APIResponse struct {
	Data []APIPrediction `json:"data"`
}
