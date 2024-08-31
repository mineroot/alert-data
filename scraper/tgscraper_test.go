package scraper_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zelenin/go-tdlib/client"
	"go.uber.org/goleak"
	"golang.org/x/sync/errgroup"

	"github.com/mineroot/alert-data/scraper"
	"github.com/mineroot/alert-data/scraper/region"
)

var kyivLocation *time.Location

func init() {
	loc, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		panic(fmt.Errorf("unable to load Europe/Kyiv timezone: %w", err))
	}
	kyivLocation = loc
}

func TestTgScraper(t *testing.T) {
	defer goleak.VerifyNone(t)

	tgScraper := scraper.NewTgScraper(
		newStubTgClient(),
		scraper.WithHistoryFromDate(strToDate("2024-08-20 00:00:00")),
		scraper.WithUpdateDiscardTimeout(0),
	)
	updates := tgScraper.UpdatesChan()

	// assert panic if nil context provided
	require.Panics(t, func() {
		_ = tgScraper.Run(nil)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return tgScraper.Run(ctx)
	})

	// wait for history scraping (for last 2 days by default)
	err := tgScraper.WaitForHistory(ctx)
	require.NoError(t, err)

	// assert alert data from history
	status, _ := tgScraper.AlertData().GetByRegion(region.Odesa)
	require.Equal(t, scraper.Status{
		Region:    region.Odesa,
		Enabled:   true,
		UpdatedAt: strToDate("2024-08-21 02:15:00"),
		IsHistory: true,
	}, status)

	// assert Crimea & Luhansk raid alert is enabled
	status, _ = tgScraper.AlertData().GetByRegion(region.Crimea)
	require.Equal(t, scraper.Status{
		Region:    region.Crimea,
		Enabled:   true,
		UpdatedAt: strToDate("2022-12-11 00:22:00"),
		IsHistory: true,
	}, status)

	status, _ = tgScraper.AlertData().GetByRegion(region.Luhansk)
	require.Equal(t, scraper.Status{
		Region:    region.Luhansk,
		Enabled:   true,
		UpdatedAt: strToDate("2022-04-04 19:45:00"),
		IsHistory: true,
	}, status)

	// assert parsed "🔴 08:39 Повітряна тривога в м. Київ ..."
	status = <-updates
	require.Equal(t, scraper.Status{
		Region:    region.KyivCity,
		Enabled:   true,
		UpdatedAt: strToDate("2024-08-22 08:39:00"),
		IsHistory: false,
	}, status)

	// assert parsed "🟢 10:06 Відбій тривоги в м. Київ. ..."
	status = <-updates
	require.Equal(t, scraper.Status{
		Region:    region.KyivCity,
		Enabled:   false,
		UpdatedAt: strToDate("2024-08-22 10:06:00"),
		IsHistory: false,
	}, status)

	// assert Run() gracefully exited with context.Canceled error
	cancel()
	err = g.Wait()
	require.ErrorIs(t, err, context.Canceled)
	// assert updates chan is closed
	_, ok := <-updates
	require.False(t, ok, "updates channel is not closed")

	// assert Run() will run only once
	// this and subsequent calls will return nil immediately
	err = tgScraper.Run(context.Background())
	require.NoError(t, err)
}

type stubTgClient struct {
	history chan *client.Message
	updates chan client.Type
}

func newStubTgClient() *stubTgClient {
	historyMessages := []*client.Message{
		createTestMessage(
			"🟢 19:46 Відбій тривоги в Одеська область.\nСлідкуйте за подальшими повідомленнями.\n#Одеська_область",
			strToDate("2024-08-19 19:46:52"),
		),
		createTestMessage(
			"🔴 02:15 Повітряна тривога в Одеська область\nСлідкуйте за подальшими повідомленнями.\n#Одеська_область",
			strToDate("2024-08-21 02:15:19"),
		),
	}
	history := make(chan *client.Message, len(historyMessages))
	defer close(history)
	slices.Reverse(historyMessages) // newer messages first
	for _, message := range historyMessages {
		history <- message
	}

	updatesMessages := []*client.Message{
		createTestMessage(
			"🔴 08:39 Повітряна тривога в м. Київ\nСлідкуйте за подальшими повідомленнями.\n#м_Київ",
			strToDate("2024-08-22 08:40:01"),
		),
		createTestMessage(
			"🟢 10:06 Відбій тривоги в м. Київ.\nСлідкуйте за подальшими повідомленнями.\n#м_Київ",
			strToDate("2024-08-22 10:06:43"),
		),
	}
	updates := make(chan client.Type, len(updatesMessages))
	//defer close(updates)
	for _, message := range updatesMessages {
		updates <- &client.UpdateNewMessage{Message: message}
	}

	return &stubTgClient{
		history: history,
		updates: updates,
	}
}

func (r *stubTgClient) GetListener() *client.Listener {
	return &client.Listener{
		Updates: r.updates,
	}
}

func (r *stubTgClient) GetChatHistory(*client.GetChatHistoryRequest) (*client.Messages, error) {
	if message, ok := <-r.history; ok {
		return &client.Messages{
			TotalCount: 1,
			Messages:   []*client.Message{message},
		}, nil
	}
	return nil, fmt.Errorf("unexpected call, set the oldest message's date to (now - 2 days)")
}

func createTestMessage(text string, date time.Time) *client.Message {
	return &client.Message{
		Date: int32(date.Unix()),
		Content: &client.MessageText{
			Text: &client.FormattedText{
				Text: text,
			},
		},
	}
}

func strToDate(dateStr string) time.Time {
	date, err := time.ParseInLocation(time.DateTime, dateStr, kyivLocation)
	if err != nil {
		panic(fmt.Errorf("failed to parse date: %s: %w", dateStr, err))
	}
	return date
}
