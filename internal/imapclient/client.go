package imapclient

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/textproto"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/dewingok/listhaul/internal/config"
)

type Client struct {
	cfg    *config.Config
	client *imapclient.Client
}

type Message struct {
	UID     uint32
	Headers textproto.MIMEHeader
}

func Connect(_ context.Context, cfg *config.Config) (*Client, error) {
	password, err := cfg.Password()
	if err != nil {
		return nil, err
	}

	address := fmt.Sprintf("%s:%d", cfg.Account.Host, cfg.Account.Port)
	c, err := imapclient.DialTLS(address, nil)
	if err != nil {
		return nil, fmt.Errorf("dial imap: %w", err)
	}

	if err := c.Login(cfg.Account.Username, password).Wait(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("login: %w", err)
	}

	if _, err := c.Select(cfg.Account.Mailbox, nil).Wait(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("select mailbox %q: %w", cfg.Account.Mailbox, err)
	}

	return &Client{cfg: cfg, client: c}, nil
}

func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}
	if err := c.client.Logout().Wait(); err != nil {
		_ = c.client.Close()
		return err
	}
	return c.client.Close()
}

func (c *Client) SearchRecent(_ context.Context) ([]uint32, error) {
	since := time.Now().Add(-c.cfg.Poll.Lookback.Duration)
	criteria := &imap.SearchCriteria{Since: since}
	if c.cfg.Poll.UnseenOnly {
		criteria.NotFlag = append(criteria.NotFlag, imap.FlagSeen)
	}

	data, err := c.client.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("uid search: %w", err)
	}

	uids := data.AllUIDs()
	result := make([]uint32, 0, len(uids))
	for _, uid := range uids {
		result = append(result, uint32(uid))
	}
	return result, nil
}

func (c *Client) FetchHeaders(_ context.Context, uids []uint32) (map[uint32]textproto.MIMEHeader, error) {
	if len(uids) == 0 {
		return map[uint32]textproto.MIMEHeader{}, nil
	}

	uidSet := imap.UIDSetNum()
	for _, uid := range uids {
		uidSet.AddNum(imap.UID(uid))
	}

	section := &imap.FetchItemBodySection{
		Specifier: imap.PartSpecifierHeader,
		Peek:      true,
	}
	options := &imap.FetchOptions{
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{section},
	}

	messages, err := c.client.Fetch(uidSet, options).Collect()
	if err != nil {
		return nil, fmt.Errorf("fetch headers: %w", err)
	}

	headers := make(map[uint32]textproto.MIMEHeader, len(messages))
	for _, msg := range messages {
		raw := msg.FindBodySection(section)
		if len(raw) == 0 {
			continue
		}
		parsed, err := textproto.NewReader(bufio.NewReader(bytes.NewReader(raw))).ReadMIMEHeader()
		if err != nil {
			return nil, fmt.Errorf("parse headers for uid %d: %w", msg.UID, err)
		}
		headers[uint32(msg.UID)] = parsed
	}
	return headers, nil
}

func (c *Client) FileInto(_ context.Context, uid uint32, folder string) error {
	uidSet := imap.UIDSetNum(imap.UID(uid))
	if _, err := c.client.Move(uidSet, folder).Wait(); err != nil {
		return fmt.Errorf("move uid %d to %q: %w", uid, folder, err)
	}
	return nil
}

func (c *Client) Discard(_ context.Context, uid uint32) error {
	uidSet := imap.UIDSetNum(imap.UID(uid))
	storeFlags := &imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: true,
		Flags:  []imap.Flag{imap.FlagDeleted},
	}
	if err := c.client.Store(uidSet, storeFlags, nil).Close(); err != nil {
		return fmt.Errorf("mark deleted uid %d: %w", uid, err)
	}
	if err := c.client.UIDExpunge(uidSet).Close(); err != nil {
		return fmt.Errorf("expunge uid %d: %w", uid, err)
	}
	return nil
}

func (c *Client) MarkRead(_ context.Context, uid uint32) error {
	return c.SetFlag(context.Background(), uid, string(imap.FlagSeen), true)
}

func (c *Client) SetFlag(_ context.Context, uid uint32, flag string, set bool) error {
	uidSet := imap.UIDSetNum(imap.UID(uid))
	op := imap.StoreFlagsAdd
	if !set {
		op = imap.StoreFlagsDel
	}
	storeFlags := &imap.StoreFlags{
		Op:     op,
		Silent: true,
		Flags:  []imap.Flag{imap.Flag(flag)},
	}
	if err := c.client.Store(uidSet, storeFlags, nil).Close(); err != nil {
		return fmt.Errorf("set flag uid %d: %w", uid, err)
	}
	return nil
}
