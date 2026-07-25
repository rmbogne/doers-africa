package handlers

import (
	"testing"

	"github.com/mbogne/african-doers/models"
	"github.com/mbogne/african-doers/store"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestSelectFeaturedEventsUsesRSVPRanking(
	t *testing.T,
) {
	firstID := primitive.NewObjectID()
	secondID := primitive.NewObjectID()
	thirdID := primitive.NewObjectID()

	upcoming := []models.Event{
		{
			ID:    firstID,
			Title: "First",
			Date:  "2026-08-10",
		},
		{
			ID:    secondID,
			Title: "Second",
			Date:  "2026-08-11",
		},
		{
			ID:    thirdID,
			Title: "Third",
			Date:  "2026-08-12",
		},
	}

	rankings := []store.EventRSVPRanking{
		{
			EventID:   secondID.Hex(),
			RSVPCount: 12,
		},
		{
			EventID:   firstID.Hex(),
			RSVPCount: 7,
		},
	}

	featured := selectFeaturedEvents(
		upcoming,
		rankings,
		3,
	)

	if len(featured) != 3 {
		t.Fatalf(
			"expected 3 events, got %d",
			len(featured),
		)
	}

	if featured[0].Event.ID != secondID ||
		featured[0].RSVPCount != 12 {
		t.Fatalf(
			"unexpected first featured event: %#v",
			featured[0],
		)
	}

	if featured[1].Event.ID != firstID ||
		featured[1].RSVPCount != 7 {
		t.Fatalf(
			"unexpected second featured event: %#v",
			featured[1],
		)
	}

	if featured[2].Event.ID != thirdID ||
		featured[2].RSVPCount != 0 {
		t.Fatalf(
			"expected unranked upcoming event as fallback: %#v",
			featured[2],
		)
	}
}

func TestSelectFeaturedEventsSkipsMissingEvents(
	t *testing.T,
) {
	eventID := primitive.NewObjectID()

	featured := selectFeaturedEvents(
		[]models.Event{
			{
				ID:    eventID,
				Title: "Visible event",
			},
		},
		[]store.EventRSVPRanking{
			{
				EventID:   primitive.NewObjectID().Hex(),
				RSVPCount: 100,
			},
		},
		1,
	)

	if len(featured) != 1 ||
		featured[0].Event.ID != eventID {
		t.Fatalf(
			"expected visible fallback event, got %#v",
			featured,
		)
	}
}
