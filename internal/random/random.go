package random

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	userMentionRe = regexp.MustCompile(`<@!?(\d+)>`)
	roleMentionRe = regexp.MustCompile(`<@&(\d+)>`)
	countRe       = regexp.MustCompile(`/(\d+)`)
)

func Handle(s *discordgo.Session, guildID, botID, input string) (string, error) {
	candidates, count, memberCache, err := ParseTargets(s, guildID, botID, input)
	if err != nil {
		return "", err
	}

	names := ResolveNames(s, guildID, candidates, memberCache)
	return Lottery(names, candidates, count), nil
}

func ParseTargets(s *discordgo.Session, guildID, botID, input string) (candidates []string, count int, memberCache []*discordgo.Member, err error) {
	count = 1
	seen := map[string]bool{}

	add := func(uid string) {
		if uid == botID || seen[uid] {
			return
		}
		seen[uid] = true
		candidates = append(candidates, uid)
	}

	needMembers := strings.Contains(input, "@everyone") || roleMentionRe.MatchString(input)
	if needMembers {
		memberCache, err = fetchAllMembers(s, guildID)
		if err != nil {
			return nil, 0, nil, fmt.Errorf("メンバー取得に失敗: %w", err)
		}
	}

	if strings.Contains(input, "@everyone") {
		for _, m := range memberCache {
			add(m.User.ID)
		}
	}

	for _, m := range roleMentionRe.FindAllStringSubmatch(input, -1) {
		roleID := m[1]
		for _, uid := range membersWithRole(memberCache, roleID) {
			add(uid)
		}
	}

	for _, m := range userMentionRe.FindAllStringSubmatch(input, -1) {
		add(m[1])
	}

	if m := countRe.FindStringSubmatch(input); len(m) > 1 {
		if n, e := strconv.Atoi(m[1]); e == nil && n > 0 {
			count = n
		}
	}

	return
}

func ResolveNames(s *discordgo.Session, guildID string, candidates []string, memberCache []*discordgo.Member) []string {
	cacheMap := make(map[string]*discordgo.Member, len(memberCache))
	for _, m := range memberCache {
		if m.User != nil {
			cacheMap[m.User.ID] = m
		}
	}

	names := make([]string, len(candidates))
	for i, id := range candidates {
		if m, ok := cacheMap[id]; ok {
			if m.Nick != "" {
				names[i] = m.Nick
			} else {
				names[i] = m.User.Username
			}
			continue
		}
		m, err := s.GuildMember(guildID, id)
		if err == nil && m.Nick != "" {
			names[i] = m.Nick
		} else if err == nil && m.User != nil {
			names[i] = m.User.Username
		} else {
			names[i] = id
		}
	}
	return names
}

func Lottery(names []string, candidates []string, count int) string {
	if len(candidates) == 0 {
		return "⚠️ 抽選対象のユーザーがいないよ"
	}
	if count > len(candidates) {
		return fmt.Sprintf("⚠️ 当選人数 (%d) が候補者数 (%d) を超えてるよ", count, len(candidates))
	}

	r := rand.New(rand.NewSource(cryptoSeed()))
	shuffled := append([]string(nil), candidates...)
	r.Shuffle(len(shuffled), func(a, b int) {
		shuffled[a], shuffled[b] = shuffled[b], shuffled[a]
	})
	winners := shuffled[:count]

	var sb strings.Builder
	sb.WriteString("🍡 **抽選結果** 🍡\n")
	sb.WriteString(fmt.Sprintf("%d 人から %d 人を選びました！\n\n", len(names), count))
	for i, id := range winners {
		sb.WriteString(fmt.Sprintf("%d. <@%s>\n", i+1, id))
	}
	sb.WriteString("\n```\n")
	sb.WriteString(fmt.Sprintf("候補者 (%d人):\n", len(names)))
	for _, name := range names {
		sb.WriteString(fmt.Sprintf("  %s\n", name))
	}
	sb.WriteString("```")
	return sb.String()
}

func fetchAllMembers(s *discordgo.Session, guildID string) ([]*discordgo.Member, error) {
	var all []*discordgo.Member
	after := ""
	for {
		members, err := s.GuildMembers(guildID, after, 1000)
		if err != nil {
			return nil, err
		}
		for _, m := range members {
			if m.User != nil && !m.User.Bot {
				all = append(all, m)
			}
		}
		if len(members) < 1000 {
			break
		}
		after = members[len(members)-1].User.ID
	}
	return all, nil
}

func membersWithRole(members []*discordgo.Member, roleID string) []string {
	var ids []string
	for _, m := range members {
		for _, r := range m.Roles {
			if r == roleID {
				ids = append(ids, m.User.ID)
				break
			}
		}
	}
	return ids
}

func cryptoSeed() int64 {
	var b [8]byte
	_, _ = cryptorand.Read(b[:])
	return int64(binary.LittleEndian.Uint64(b[:]))
}
