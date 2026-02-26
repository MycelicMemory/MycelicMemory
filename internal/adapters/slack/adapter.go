package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MycelicMemory/mycelicmemory/internal/logging"
	"github.com/MycelicMemory/mycelicmemory/internal/pipeline"
)

var log = logging.GetLogger("slack-adapter")

// Adapter implements pipeline.SourceAdapter for Slack workspace exports.
type Adapter struct {
	config     Config
	checkpoint string

	// Cached export data
	users    map[string]*User
	channels []Channel
}

// NewAdapter creates a new Slack adapter.
func NewAdapter() *Adapter {
	return &Adapter{
		users: make(map[string]*User),
	}
}

// Type returns the source type identifier.
func (a *Adapter) Type() string {
	return "slack"
}

// Configure initializes the adapter with Slack-specific config.
func (a *Adapter) Configure(config json.RawMessage) error {
	a.config = Config{
		MinMessages: 2,
	}
	if err := json.Unmarshal(config, &a.config); err != nil {
		return fmt.Errorf("invalid slack config: %w", err)
	}
	if a.config.ExportPath == "" {
		return fmt.Errorf("export_path is required")
	}
	return nil
}

// Validate checks that the export directory exists and contains expected files.
func (a *Adapter) Validate() error {
	info, err := os.Stat(a.config.ExportPath)
	if err != nil {
		return fmt.Errorf("export path not accessible: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("export_path must be a directory: %s", a.config.ExportPath)
	}

	// Check for users.json
	usersPath := filepath.Join(a.config.ExportPath, "users.json")
	if _, err := os.Stat(usersPath); err != nil {
		return fmt.Errorf("users.json not found in export: %w", err)
	}

	// Check for channels.json
	channelsPath := filepath.Join(a.config.ExportPath, "channels.json")
	if _, err := os.Stat(channelsPath); err != nil {
		return fmt.Errorf("channels.json not found in export: %w", err)
	}

	return nil
}

// Checkpoint returns the current position for resume.
func (a *Adapter) Checkpoint() string {
	return a.checkpoint
}

// ReadItems streams ConversationItems from the Slack export.
func (a *Adapter) ReadItems(ctx context.Context, checkpoint string) (<-chan pipeline.ConversationItem, <-chan error) {
	items := make(chan pipeline.ConversationItem, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(items)
		defer close(errCh)

		if err := a.loadMetadata(); err != nil {
			errCh <- fmt.Errorf("failed to load slack metadata: %w", err)
			return
		}

		log.Info("loaded slack export metadata",
			"users", len(a.users),
			"channels", len(a.channels),
		)

		// Parse checkpoint: "channel_idx:msg_ts" format
		checkpointChannelIdx := 0
		checkpointTs := ""
		if checkpoint != "" {
			parts := strings.SplitN(checkpoint, ":", 2)
			if len(parts) == 2 {
				checkpointChannelIdx, _ = strconv.Atoi(parts[0])
				checkpointTs = parts[1]
			}
		}

		// Process each channel
		for i, channel := range a.channels {
			if i < checkpointChannelIdx {
				continue
			}

			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			// Filter channels if configured
			if len(a.config.Channels) > 0 {
				found := false
				for _, name := range a.config.Channels {
					if channel.Name == name || channel.ID == name {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			// Skip private if not configured
			if channel.IsPrivate && !a.config.IncludePrivate {
				continue
			}

			channelItems, err := a.readChannelMessages(ctx, channel, i, checkpointTs)
			if err != nil {
				log.Warn("failed to read channel", "channel", channel.Name, "error", err)
				continue
			}

			// Reset checkpoint for subsequent channels
			if i == checkpointChannelIdx {
				checkpointTs = ""
			}

			for _, item := range channelItems {
				select {
				case items <- item:
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				}
			}
		}
	}()

	return items, errCh
}

// loadMetadata loads users.json and channels.json
func (a *Adapter) loadMetadata() error {
	// Load users
	usersPath := filepath.Join(a.config.ExportPath, "users.json")
	usersData, err := os.ReadFile(usersPath)
	if err != nil {
		return fmt.Errorf("failed to read users.json: %w", err)
	}
	var users []User
	if err := json.Unmarshal(usersData, &users); err != nil {
		return fmt.Errorf("failed to parse users.json: %w", err)
	}
	for i := range users {
		a.users[users[i].ID] = &users[i]
	}

	// Load channels
	channelsPath := filepath.Join(a.config.ExportPath, "channels.json")
	channelsData, err := os.ReadFile(channelsPath)
	if err != nil {
		return fmt.Errorf("failed to read channels.json: %w", err)
	}
	if err := json.Unmarshal(channelsData, &a.channels); err != nil {
		return fmt.Errorf("failed to parse channels.json: %w", err)
	}

	// Also load groups.json (private channels) if it exists and configured
	if a.config.IncludePrivate {
		groupsPath := filepath.Join(a.config.ExportPath, "groups.json")
		if groupsData, err := os.ReadFile(groupsPath); err == nil {
			var groups []Channel
			if err := json.Unmarshal(groupsData, &groups); err == nil {
				for i := range groups {
					groups[i].IsPrivate = true
				}
				a.channels = append(a.channels, groups...)
			}
		}
	}

	// Also load DMs if configured
	if a.config.IncludeDMs {
		dmsPath := filepath.Join(a.config.ExportPath, "dms.json")
		if dmsData, err := os.ReadFile(dmsPath); err == nil {
			var dms []Channel
			if err := json.Unmarshal(dmsData, &dms); err == nil {
				for i := range dms {
					dms[i].IsPrivate = true
				}
				a.channels = append(a.channels, dms...)
			}
		}

		// Group DMs
		mpimsPath := filepath.Join(a.config.ExportPath, "mpims.json")
		if mpimsData, err := os.ReadFile(mpimsPath); err == nil {
			var mpims []Channel
			if err := json.Unmarshal(mpimsData, &mpims); err == nil {
				for i := range mpims {
					mpims[i].IsPrivate = true
				}
				a.channels = append(a.channels, mpims...)
			}
		}
	}

	return nil
}

// readChannelMessages reads all messages from a channel directory
func (a *Adapter) readChannelMessages(ctx context.Context, channel Channel, channelIdx int, afterTs string) ([]pipeline.ConversationItem, error) {
	channelDir := filepath.Join(a.config.ExportPath, channel.Name)
	if _, err := os.Stat(channelDir); err != nil {
		// Try channel ID as directory name (some exports use IDs)
		channelDir = filepath.Join(a.config.ExportPath, channel.ID)
		if _, err := os.Stat(channelDir); err != nil {
			return nil, nil // No messages for this channel
		}
	}

	// Find all date JSON files in the channel directory
	dateFiles, err := filepath.Glob(filepath.Join(channelDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob channel directory: %w", err)
	}

	if len(dateFiles) == 0 {
		return nil, nil
	}

	// Sort date files chronologically
	sort.Strings(dateFiles)

	// Read all messages from all date files
	var allMessages []Message
	for _, dateFile := range dateFiles {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		data, err := os.ReadFile(dateFile)
		if err != nil {
			log.Warn("failed to read date file", "file", dateFile, "error", err)
			continue
		}

		var messages []Message
		if err := json.Unmarshal(data, &messages); err != nil {
			log.Warn("failed to parse date file", "file", dateFile, "error", err)
			continue
		}

		allMessages = append(allMessages, messages...)
	}

	if len(allMessages) < a.config.MinMessages {
		return nil, nil
	}

	// Sort messages by timestamp
	sort.Slice(allMessages, func(i, j int) bool {
		return allMessages[i].Ts < allMessages[j].Ts
	})

	// Filter messages after checkpoint
	startIdx := 0
	if afterTs != "" {
		for i, msg := range allMessages {
			if msg.Ts > afterTs {
				startIdx = i
				break
			}
		}
	}

	// Convert to ConversationItems
	var items []pipeline.ConversationItem
	for i := startIdx; i < len(allMessages); i++ {
		msg := allMessages[i]

		// Skip system/meta messages
		if shouldSkipMessage(msg) {
			continue
		}

		item := a.convertMessage(msg, channel, i)
		items = append(items, item)

		// Update checkpoint
		a.checkpoint = fmt.Sprintf("%d:%s", channelIdx, msg.Ts)
	}

	return items, nil
}

// convertMessage converts a Slack message to a ConversationItem
func (a *Adapter) convertMessage(msg Message, channel Channel, seqIdx int) pipeline.ConversationItem {
	item := pipeline.ConversationItem{
		ExternalID:     fmt.Sprintf("slack-%s-%s", channel.ID, msg.Ts),
		SourceType:     "slack",
		ConversationID: channel.ID,
		ProjectOrSpace: channel.Name,
		Content:        a.resolveUserMentions(msg.Text),
		ContentType:    "text",
		Timestamp:      parseSlackTimestamp(msg.Ts),
		SequenceIndex:  seqIdx,
		Metadata:       make(map[string]any),
	}

	// Determine role and author
	if msg.BotID != "" || msg.BotProfile != nil {
		item.Role = "bot"
		if msg.BotProfile != nil {
			item.Author = msg.BotProfile.Name
		} else {
			item.Author = "bot"
		}
	} else if user, ok := a.users[msg.User]; ok {
		item.Role = "user"
		item.Author = userDisplayName(user)
	} else {
		item.Role = "user"
		item.Author = msg.User
	}

	// Thread info
	if msg.ThreadTs != "" && msg.ThreadTs != msg.Ts {
		item.ThreadID = msg.ThreadTs
		item.ReplyToID = msg.ThreadTs
	}

	// Store metadata
	item.Metadata["channel_name"] = channel.Name
	item.Metadata["channel_id"] = channel.ID
	if channel.IsPrivate {
		item.Metadata["is_private"] = true
	}
	if msg.SubType != "" {
		item.Metadata["subtype"] = msg.SubType
	}

	// Convert file attachments
	for _, file := range msg.Files {
		att := pipeline.Attachment{
			Name:     file.Name,
			URL:      file.Permalink,
			MimeType: file.Mimetype,
		}
		if file.Mimetype != "" && strings.HasPrefix(file.Mimetype, "image/") {
			att.Type = "image"
		} else {
			att.Type = "file"
		}
		item.Attachments = append(item.Attachments, att)
	}

	// Convert reactions to actions
	for _, reaction := range msg.Reactions {
		action := pipeline.Action{
			Type:      "reaction",
			Name:      reaction.Name,
			Success:   true,
			Timestamp: item.Timestamp,
		}
		// Store reacting user count in input
		action.Input = fmt.Sprintf(`{"count":%d,"users":%d}`, reaction.Count, len(reaction.Users))
		item.Actions = append(item.Actions, action)
	}

	return item
}

// resolveUserMentions replaces <@U123> with @username
func (a *Adapter) resolveUserMentions(text string) string {
	result := text
	for id, user := range a.users {
		mention := fmt.Sprintf("<@%s>", id)
		replacement := "@" + userDisplayName(user)
		result = strings.ReplaceAll(result, mention, replacement)
	}
	return result
}

// shouldSkipMessage returns true for system messages that shouldn't be ingested
func shouldSkipMessage(msg Message) bool {
	switch msg.SubType {
	case "channel_join", "channel_leave", "channel_topic", "channel_purpose",
		"channel_name", "channel_archive", "message_deleted",
		"pinned_item", "unpinned_item":
		return true
	}
	// Skip empty messages
	if msg.Text == "" && len(msg.Files) == 0 {
		return true
	}
	return false
}

// parseSlackTimestamp converts "1355517523.000005" to time.Time
func parseSlackTimestamp(ts string) time.Time {
	parts := strings.SplitN(ts, ".", 2)
	if len(parts) == 0 {
		return time.Time{}
	}
	secs, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}
	}
	var nsecs int64
	if len(parts) > 1 && parts[1] != "" {
		// Pad to 9 digits for nanoseconds
		micro := parts[1]
		for len(micro) < 9 {
			micro += "0"
		}
		nsecs, _ = strconv.ParseInt(micro[:9], 10, 64)
	}
	return time.Unix(secs, nsecs)
}

// userDisplayName returns the best display name for a user
func userDisplayName(user *User) string {
	if user.Profile.DisplayName != "" {
		return user.Profile.DisplayName
	}
	if user.Profile.RealName != "" {
		return user.Profile.RealName
	}
	return user.Name
}
