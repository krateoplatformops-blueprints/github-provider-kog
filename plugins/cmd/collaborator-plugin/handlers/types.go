package collaborator

type RepoPermissions struct {
	HTMLURL     string      `json:"html_url"`
	ID          int         `json:"id"` // user ID
	Permission  string      `json:"permission"`
	Permissions Permissions `json:"permissions"`
	RoleName    string      `json:"role_name"`
	User        User        `json:"user"`
	Message     string      `json:"message"`
}

type User struct {
	AvatarURL         string      `json:"avatar_url"`
	EventsURL         string      `json:"events_url"`
	FollowersURL      string      `json:"followers_url"`
	FollowingURL      string      `json:"following_url"`
	GistsURL          string      `json:"gists_url"`
	GravatarID        string      `json:"gravatar_id"`
	HTMLURL           string      `json:"html_url"`
	ID                int         `json:"id"`
	Login             string      `json:"login"`
	NodeID            string      `json:"node_id"`
	OrganizationsURL  string      `json:"organizations_url"`
	Permissions       Permissions `json:"permissions"`
	ReceivedEventsURL string      `json:"received_events_url"`
	ReposURL          string      `json:"repos_url"`
	RoleName          string      `json:"role_name"`
	SiteAdmin         bool        `json:"site_admin"`
	StarredURL        string      `json:"starred_url"`
	SubscriptionsURL  string      `json:"subscriptions_url"`
	Type              string      `json:"type"`
	URL               string      `json:"url"`
	UserViewType      string      `json:"user_view_type"`
}

type Permissions struct {
	Admin    bool `json:"admin"`
	Maintain bool `json:"maintain"`
	Push     bool `json:"push"`
	Triage   bool `json:"triage"`
	Pull     bool `json:"pull"`
}

type Message struct {
	Message string `json:"message"`
}

type Permission struct {
	Permission string `json:"permission"`
}
