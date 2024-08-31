package scraper

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"sync"
	"time"

	"github.com/zelenin/go-tdlib/client"
	"golang.org/x/sync/errgroup"

	"github.com/mineroot/alert-data/scraper/region"
)

const airAlertUaChannelID int64 = -1001766138888

var alertStatusRegexp = regexp.MustCompile(`(?m)^[🔴🟢🟡] (\d\d:\d\d) (Відбій тривоги|Повітряна тривога) в (.*?)\.?$`)

// TgScraper is a struct that handles scraping alert status updates from a Telegram channel.
// It provides methods to run the scraper, retrieve alert data, and get real-time status updates.
type TgScraper struct {
	client               TgClient
	historyFromDate      time.Time
	updateDiscardTimeout time.Duration

	once        sync.Once
	historyDone chan struct{}
	alertData   *AlertData
	updates     chan Status
}

// NewTgScraper creates a TgScraper with the given TgClient and optional settings.
func NewTgScraper(client TgClient, opts ...func(*TgScraper)) *TgScraper {
	scraper := &TgScraper{
		client:               client,
		historyFromDate:      time.Now().Add(-2 * 24 * time.Hour), // 2 days ago
		updateDiscardTimeout: 0,

		once:        sync.Once{},
		historyDone: make(chan struct{}),
		alertData:   newAlertData(),
		updates:     nil,
	}
	for _, o := range opts {
		o(scraper)
	}
	return scraper
}

// WithHistoryFromDate sets the date from which to start fetching history.
// Default is the date 2 days ago.
func WithHistoryFromDate(historyFromDate time.Time) func(*TgScraper) {
	return func(s *TgScraper) {
		s.historyFromDate = historyFromDate
	}
}

// WithUpdateDiscardTimeout sets the timeout for discarding updates if UpdateChan() is full.
// Default is 0, meaning updates won't be discarded, but the whole processing may be blocked if receiver is too slow.
func WithUpdateDiscardTimeout(timeout time.Duration) func(*TgScraper) {
	return func(s *TgScraper) {
		s.updateDiscardTimeout = timeout
	}
}

// Run starts the scraper.
func (r *TgScraper) Run(ctx context.Context) error {
	if r.client == nil {
		panic("scraper: use scraper.NewTgScraper() to create *TgScraper instance")
	}
	if ctx == nil {
		panic("scraper: nil context")
	}
	var err error
	r.once.Do(func() {
		err = r.run(ctx)
	})

	if err != nil {
		return fmt.Errorf("scraper: %w", err)
	}
	return nil
}

// WaitForHistory blocks until historical data has been fetched.
func (r *TgScraper) WaitForHistory(ctx context.Context) error {
	select {
	case <-r.historyDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// AlertData returns current alert statuses.
func (r *TgScraper) AlertData() *AlertData {
	return r.alertData
}

// UpdatesChan returns a channel with real-time status updates.
func (r *TgScraper) UpdatesChan() <-chan Status {
	if r.updates == nil {
		r.updates = make(chan Status, 1)
	}
	return r.updates
}

func (r *TgScraper) run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return r.history(ctx)
	})
	g.Go(func() error {
		return r.listenUpdates(ctx)
	})

	return g.Wait()
}

func (r *TgScraper) history(ctx context.Context) error {
	defer close(r.historyDone)
	messages, err := r.getMessagesForPeriod(ctx, r.historyFromDate)
	if err != nil {
		return err
	}
	slices.Reverse(messages) // reverse slice so first message is most old
	for _, message := range messages {
		status, err := r.parseMessage(message)
		if err != nil {
			return fmt.Errorf("unable to scrape history: %w", err)
		}
		if status == nil {
			continue
		}
		status.IsHistory = true

		r.alertData.set(status)
	}

	return nil
}

func (r *TgScraper) listenUpdates(ctx context.Context) error {
	defer r.closeUpdates()

	listener := r.client.GetListener()
	defer listener.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-listener.Updates:
			if update == nil {
				return fmt.Errorf("received nil update")
			}
			// todo check this message from desired channel
			if update.GetType() != client.TypeUpdateNewMessage {
				break
			}
			updateNewMessage, _ := update.(*client.UpdateNewMessage)
			status, err := r.parseMessage(updateNewMessage.Message)
			if err != nil {
				return fmt.Errorf("unable to scrape update: %w", err)
			}
			if status == nil {
				break
			}
			r.alertData.set(status)
			r.sendUpdate(ctx, *status)
		}
	}
}

func (r *TgScraper) sendUpdate(ctx context.Context, status Status) {
	if r.updates == nil {
		return
	}
	if r.updateDiscardTimeout != 0 {
		var cancel context.CancelFunc = func() {}
		ctx, cancel = context.WithTimeout(ctx, r.updateDiscardTimeout)
		defer cancel()
	}
	select {
	case <-ctx.Done():
	case r.updates <- status:
	}
}

// getMessagesForPeriod returns history for period (from now to now-period)
func (r *TgScraper) getMessagesForPeriod(ctx context.Context, historyFromDate time.Time) ([]*client.Message, error) {
	messagesForPeriod := make([]*client.Message, 0, 200)
	fromMessageId := int64(0)
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		messages, err := r.client.GetChatHistory(&client.GetChatHistoryRequest{
			ChatId:        airAlertUaChannelID,
			FromMessageId: fromMessageId,
			Offset:        0,
			Limit:         1, // tdLib always returns one message no matter what limit is
			OnlyLocal:     false,
		})
		if err != nil {
			return nil, err
		}
		if len(messages.Messages) == 0 {
			break // no history left (should be unreachable in airAlertUaChannelID channel)
		}
		message := messages.Messages[0]
		messageDate := time.Unix(int64(message.Date), 0)
		if messageDate.Before(historyFromDate) {
			break // to old
		}
		fromMessageId = message.Id

		if message.ForwardInfo != nil {
			continue // skip forwarded posts
		}

		if message.Content.MessageContentType() != client.TypeMessageText {
			continue // skip not text messages
		}
		messagesForPeriod = append(messagesForPeriod, message)
	}
	return messagesForPeriod, nil
}

func (r *TgScraper) parseMessage(message *client.Message) (*Status, error) {
	messageText, ok := message.Content.(*client.MessageText)
	if !ok {
		return nil, nil
	}
	messageTextStr := messageText.Text.Text

	match := alertStatusRegexp.FindStringSubmatch(messageTextStr)
	if len(match) < 3 {
		return nil, nil
	}

	messageAt := time.Unix(int64(message.Date), 0)
	timeOnly := match[1] + ":00"
	parsedTime, err := time.Parse(time.TimeOnly, timeOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time: %s: %w", timeOnly, err)
	}
	updatedAt := time.Date(
		messageAt.Year(), messageAt.Month(), messageAt.Day(),
		parsedTime.Hour(), parsedTime.Minute(),
		0, 0, kyivLocation,
	)
	// in rare case when message arrives at 00:01 but parsed time is 23:59
	if updatedAt.After(messageAt) {
		updatedAt.Add(-24 * time.Hour)
	}

	raidStatusStr := match[2]
	var raidEnabled bool
	switch raidStatusStr {
	case "Відбій тривоги":
		raidEnabled = false
	case "Повітряна тривога":
		raidEnabled = true
	default:
		return nil, nil
	}

	regionStr := match[3]
	regionId, err := region.Parse(regionStr)
	if err != nil {
		return nil, nil
	}

	return &Status{
		Region:    regionId,
		Enabled:   raidEnabled,
		UpdatedAt: updatedAt,
	}, nil
}

func (r *TgScraper) closeUpdates() {
	if r.updates != nil {
		close(r.updates)
	}
}
