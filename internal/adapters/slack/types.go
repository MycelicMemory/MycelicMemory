package slack

// Slack export JSON structures

// Channel represents a channel from channels.json or groups.json
type Channel struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Created    int64        `json:"created"`
	Creator    string       `json:"creator"`
	IsArchived bool         `json:"is_archived"`
	IsGeneral  bool         `json:"is_general"`
	IsPrivate  bool         `json:"is_private"`
	IsMPIM     bool         `json:"is_mpim"`
	Members    []string     `json:"members"`
	Topic      ChannelTopic `json:"topic"`
	Purpose    ChannelTopic `json:"purpose"`
}

// ChannelTopic represents channel topic or purpose
type ChannelTopic struct {
	Value   string `json:"value"`
	Creator string `json:"creator"`
	LastSet int64  `json:"last_set"`
}

// User represents a user from users.json
type User struct {
	ID      string      `json:"id"`
	TeamID  string      `json:"team_id"`
	Name    string      `json:"name"`
	Deleted bool        `json:"deleted"`
	Profile UserProfile `json:"profile"`
	IsBot   bool        `json:"is_bot"`
	IsAdmin bool        `json:"is_admin"`
}

// UserProfile contains user profile information
type UserProfile struct {
	RealName    string `json:"real_name"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Title       string `json:"title"`
	StatusText  string `json:"status_text"`
	StatusEmoji string `json:"status_emoji"`
}

// Message represents a single message from a channel export file
type Message struct {
	Type            string     `json:"type"`
	SubType         string     `json:"subtype,omitempty"`
	User            string     `json:"user,omitempty"`
	BotID           string     `json:"bot_id,omitempty"`
	Text            string     `json:"text"`
	Ts              string     `json:"ts"`
	ClientMsgID     string     `json:"client_msg_id,omitempty"`
	ThreadTs        string     `json:"thread_ts,omitempty"`
	ParentUserID    string     `json:"parent_user_id,omitempty"`
	ReplyCount      int        `json:"reply_count,omitempty"`
	ReplyUsersCount int        `json:"reply_users_count,omitempty"`
	LatestReply     string     `json:"latest_reply,omitempty"`
	Reactions       []Reaction `json:"reactions,omitempty"`
	Files           []File     `json:"files,omitempty"`
	Upload          bool       `json:"upload,omitempty"`
	BotProfile      *BotInfo   `json:"bot_profile,omitempty"`
}

// Reaction represents an emoji reaction on a message
type Reaction struct {
	Name  string   `json:"name"`
	Count int      `json:"count"`
	Users []string `json:"users"`
}

// File represents an attached file
type File struct {
	ID                 string `json:"id"`
	Created            int64  `json:"created"`
	Name               string `json:"name"`
	Title              string `json:"title"`
	Mimetype           string `json:"mimetype"`
	Filetype           string `json:"filetype"`
	PrettyType         string `json:"pretty_type"`
	User               string `json:"user"`
	Size               int    `json:"size"`
	URLPrivate         string `json:"url_private"`
	URLPrivateDownload string `json:"url_private_download"`
	Permalink          string `json:"permalink"`
}

// BotInfo represents bot metadata
type BotInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Config holds adapter configuration
type Config struct {
	ExportPath     string   `json:"export_path"`
	IncludePrivate bool     `json:"include_private"`
	IncludeDMs     bool     `json:"include_dms"`
	Channels       []string `json:"channels,omitempty"` // Empty = all channels
	MinMessages    int      `json:"min_messages"`
}
