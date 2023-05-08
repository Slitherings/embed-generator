package access

import (
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/merlinfuchs/discordgo"
	"github.com/merlinfuchs/embed-generator/embedg-server/bot"
)

type AccessManager struct {
	bot         *bot.Bot
	memberCache *ttlcache.Cache[string, *discordgo.Member]
}

func New(bot *bot.Bot) *AccessManager {
	memberCache := ttlcache.New(
		ttlcache.WithTTL[string, *discordgo.Member](3*time.Minute),
		ttlcache.WithDisableTouchOnHit[string, *discordgo.Member](),
	)
	go memberCache.Start()

	return &AccessManager{
		bot:         bot,
		memberCache: memberCache,
	}
}

type GuildAccess struct {
	HasChannelWithUserAccess bool
	HasChannelWithBotAccess  bool
}

type ChannelAccess struct {
	UserPermissions int64
	BotPermissions  int64
}

func (c *ChannelAccess) UserAccess() bool {
	return c.UserPermissions&discordgo.PermissionManageWebhooks != 0
}

func (c *ChannelAccess) BotAccess() bool {
	return c.BotPermissions&discordgo.PermissionManageWebhooks != 0
}

func (m *AccessManager) GetGuildAccessForUser(userID string, guildID string) (GuildAccess, error) {
	res := GuildAccess{}
	mu := &sync.Mutex{}

	guild, err := m.bot.State.Guild(guildID)
	if err != nil {
		if err == discordgo.ErrStateNotFound {
			return res, nil
		}
		return res, err
	}

	for _, channel := range guild.Channels {
		access, err := m.GetChannelAccessForUser(userID, channel.ID)
		if err != nil {
			return res, err
		}

		mu.Lock()
		if access.BotAccess() {
			res.HasChannelWithBotAccess = true
		}
		if access.UserAccess() {
			res.HasChannelWithUserAccess = true
		}
		mu.Unlock()
	}

	return res, nil
}

func (m *AccessManager) GetChannelAccessForUser(userID string, channelID string) (ChannelAccess, error) {
	res := ChannelAccess{}

	botPerms, err := m.ComputeBotPermissionsForChannel(channelID)
	if err != nil {
		return res, err
	}
	if botPerms == 0 {
		// The bot doesn't have access to the server so there is no point in checking access for the user
		return res, nil
	}
	res.BotPermissions = botPerms

	res.UserPermissions, err = m.ComputeUserPermissionsForChannel(userID, channelID)
	if err != nil {
		return res, err
	}

	return res, nil
}

func (m *AccessManager) ComputeUserPermissionsForChannel(userID string, channelID string) (int64, error) {
	// We need to make sure the member is in the state, otherwise the permissions will be wrong
	channel, err := m.bot.State.Channel(channelID)
	if err != nil {
		if err == discordgo.ErrStateNotFound {
			return 0, nil
		}
		return 0, err
	}

	guild, err := m.bot.State.Guild(channel.GuildID)
	if err != nil {
		if err == discordgo.ErrStateNotFound {
			return 0, nil
		}
		return 0, err
	}

	member, err := m.GetGuildMember(guild.ID, userID)
	if err != nil {
		return 0, err
	}

	// this is workaround to compute the permissions using discordgo, we remove it afterwards
	m.bot.State.MemberAdd(member)

	perms, err := m.bot.State.UserChannelPermissions(userID, channelID)
	if err == discordgo.ErrStateNotFound {
		return 0, nil
	}

	m.bot.State.MemberRemove(member)

	return perms, err
}

func (m *AccessManager) ComputeBotPermissionsForChannel(channelID string) (int64, error) {
	perms, err := m.bot.State.UserChannelPermissions(m.bot.State.User.ID, channelID)
	if err == discordgo.ErrStateNotFound {
		return 0, nil
	}
	return perms, err
}

func (m *AccessManager) GetGuildMember(guildID string, userID string) (*discordgo.Member, error) {
	cacheKey := guildID + userID
	cacheItem := m.memberCache.Get(cacheKey)
	if cacheItem != nil {
		return cacheItem.Value(), nil
	}

	member, err := m.bot.Session.GuildMember(guildID, userID)
	if err != nil {
		return nil, err
	}

	m.memberCache.Set(cacheKey, member, time.Minute*3)
	return member, nil
}