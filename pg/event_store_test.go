package pg

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/findrandomevents/eventdb"
	"github.com/findrandomevents/eventdb/errors"
	"github.com/findrandomevents/eventdb/geojson"
	"github.com/findrandomevents/eventdb/pg/pgtest"

	"github.com/go-test/deep"
)

func TestEventSave(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbx := pgtest.NewDB(t)
	eventStore := &EventStore{DB: dbx}
	if err := eventStore.Init(ctx); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		Name  string
		Input string
		Want  eventdb.Event
	}{
		{
			Name: "valid event",
			Input: `{
				"id": "99999",
				"name": "Some event",
				"description": "Some Description",
				"place": {
					"name": "A place",
					"location": {
						"latitude": 20,
						"longitude": -20
					}
				},
				"start_time": "2017-05-17T17:00:00+0200",
				"end_time": "2017-05-17T20:00:00+0200",
				"is_canceled": true,
				"cover": {
					"source": "http://example.com/cover.jpg"
				}
			}`,
			Want: eventdb.Event{
				ID:          "99999",
				Name:        "Some event",
				Description: "Some Description",
				Place:       "A place",
				Latitude:    20,
				Longitude:   -20,
				StartTime:   time.Date(2017, 5, 17, 15, 0, 0, 0, time.UTC),
				EndTime:     time.Date(2017, 5, 17, 18, 0, 0, 0, time.UTC),
				IsCanceled:  true,

				Cover: "http://example.com/cover.jpg",
			},
		},
		{
			Name: "no place name",
			Input: `{
				"id": "111",
				"place": {
					"location": {
						"street": "street address"
					}
				},
				"start_time": "2017-05-17T17:00:00+0200",
				"end_time": "2017-05-17T20:00:00+0200"
			}`,
			Want: eventdb.Event{
				ID:        "111",
				Address:   "street address",
				StartTime: time.Date(2017, 5, 17, 15, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2017, 5, 17, 18, 0, 0, 0, time.UTC),
			},
		},
		{
			Name: "no end time",
			Input: `{
				"id": "222",
				"start_time": "2017-05-17T17:00:00+0200"
			}`,
			Want: eventdb.Event{
				ID:        "222",
				StartTime: time.Date(2017, 5, 17, 15, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2017, 5, 17, 16, 0, 0, 0, time.UTC),
			},
		},
		{
			Name: "time zone",
			Input: `{
				"id": "333",
				"timezone": "Europe/Belgrade",
				"start_time": "2017-05-17T17:00:00+0200",
				"end_time": "2017-05-17T20:00:00+0200"
			}`,
			Want: eventdb.Event{
				ID:        "333",
				StartTime: time.Date(2017, 5, 17, 17, 0, 0, 0, getTZ("Europe/Belgrade")),
				EndTime:   time.Date(2017, 5, 17, 20, 0, 0, 0, getTZ("Europe/Belgrade")),
			},
		},
	} {
		event, err := eventStore.Save(ctx, json.RawMessage(test.Input))
		if err != nil {
			t.Fatalf("save event (%s): %v", test.Name, err)
		}

		if diff := deep.Equal(event, test.Want); diff != nil {
			t.Fatalf("save event (%s): %v", test.Name, diff)
		}

		_, wantOffset := test.Want.StartTime.Zone()
		_, gotOffset := event.StartTime.Zone()
		if wantOffset != gotOffset {
			t.Fatalf("save event (%s): startTime offset = %d, want %d", test.Name, gotOffset, wantOffset)
		}

		_, wantOffset = test.Want.EndTime.Zone()
		_, gotOffset = event.EndTime.Zone()
		if wantOffset != gotOffset {
			t.Fatalf("save event (%s): endTime offset = %d, want %d", test.Name, gotOffset, wantOffset)
		}
	}
}

func TestSetBad(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbx := pgtest.NewDB(t)
	eventStore := &EventStore{DB: dbx}
	if err := eventStore.Init(ctx); err != nil {
		t.Fatal(err)
	}

	saved, err := eventStore.Save(ctx, json.RawMessage(`{
			"id": "99999",
			"name": "Some event",
			"description": "Some Description",
			"place": {
				"name": "A place",
				"location": {
					"latitude": 20,
					"longitude": -20
				}
			},
			"start_time": "2017-05-17T17:00:00+0200",
			"end_time": "2017-05-17T20:00:00+0200",
			"is_canceled": true,
			"cover": {
				"source": "http://example.com/cover.jpg"
			}
		}`))
	if err != nil {
		t.Fatalf("save event: %v", err)
	}
	if got, want := saved.IsBad, false; want != got {
		t.Fatalf("before SetBad(), bad = %v, want %v", got, want)
	}

	if err = eventStore.SetBad(ctx, saved.ID, true); err != nil {
		t.Fatalf("SetBad: %v", err)
	}

	updated, err := eventStore.GetByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got, want := updated.IsBad, true; got != want {
		t.Fatalf("after SetBad(): bad = %v, want %v", got, want)
	}

	if err = eventStore.SetBad(ctx, saved.ID, false); err != nil {
		t.Fatalf("SetBad: %v", err)
	}
	reverted, err := eventStore.GetByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got, want := reverted.IsBad, false; got != want {
		t.Fatalf("after SetBad(): bad = %v, want %v", got, want)
	}
}
func TestEventGet(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbx := pgtest.NewDB(t)
	eventStore := &EventStore{DB: dbx}
	if err := eventStore.Init(ctx); err != nil {
		t.Fatal(err)
	}

	saved, err := eventStore.Save(ctx, json.RawMessage(`{
		"id": "99999",
		"name": "Some event",
		"description": "Some Description",
		"place": {
			"name": "A place",
			"location": {
				"latitude": 20,
				"longitude": -20
			}
		},
		"start_time": "2017-05-17T17:00:00+0200",
		"end_time": "2017-05-17T20:00:00+0200",
		"is_canceled": true,
		"cover": {
			"source": "http://example.com/cover.jpg"
		}
	}`))
	if err != nil {
		t.Fatalf("save event: %v", err)
	}

	for _, test := range []struct {
		ID        eventdb.EventID
		WantEvent eventdb.Event
		WantErr   error
	}{
		{
			ID:      "nonexistent",
			WantErr: errors.E(errors.NotExist),
		},
		{
			ID:        saved.ID,
			WantEvent: saved,
			WantErr:   nil,
		},
	} {
		event, err := eventStore.GetByID(ctx, test.ID)
		if test.WantErr == nil {
			if got, want := err, test.WantErr; got != want {
				t.Fatalf("get event: err=%v, want %v", got, want)
			}
		} else if got, want := err, test.WantErr; !errors.Match(got, want) {
			t.Fatalf("get event: err=%v, want %v", got, want)
		}

		if diff := deep.Equal(event, test.WantEvent); diff != nil {
			t.Fatalf("get event: %v", diff)
		}
	}
}

func TestEventSearchFilter(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, test := range []struct {
		Name    string
		Events  []string
		IsBad   bool
		Search  eventdb.EventSearchRequest
		WantIDs []eventdb.EventID
	}{
		{
			Name: "in bounds",
			Events: []string{`{
				"id": "1",
				"start_time": "2000-01-01T00:00:00Z",
				"place": {
					"location": {
						"street": "street addr",
						"latitude": 20,
						"longitude": 20
					}
				}
			}`},
			Search: eventdb.EventSearchRequest{
				Bounds: geojson.CircleGeom(20, 20, 1),
				Start:  time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
				End:    time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			WantIDs: []eventdb.EventID{"1"},
		},
		{
			Name: "out of in bounds",
			Events: []string{`{
				"id": "1",
				"start_time": "2000-01-01T00:00:00Z",
				"place": {
					"location": {
						"street": "street addr",
						"latitude": 20,
						"longitude": 20
					}
				}
			}`},
			Search: eventdb.EventSearchRequest{
				Bounds: geojson.CircleGeom(0, 0, 1),
				Start:  time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
				End:    time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			WantIDs: nil,
		},
		{
			Name: "no street address",
			Events: []string{`{
				"id": "1",
				"start_time": "2000-01-01T00:00:00Z",
				"place": {
					"location": {
						"latitude": 20,
						"longitude": 20
					}
				}
			}`},
			Search: eventdb.EventSearchRequest{
				Bounds: geojson.CircleGeom(20, 20, 1),
				Start:  time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
				End:    time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			WantIDs: nil,
		},
		{
			Name: "long event",
			Events: []string{`{
				"id": "1",
				"start_time": "2000-01-01T00:00:00Z",
				"end_time": "2001-01-01T00:00:00Z",
				"place": {
					"location": {
						"street": "street addr",
						"latitude": 20,
						"longitude": 20
					}
				}
			}`},
			Search: eventdb.EventSearchRequest{
				Bounds: geojson.CircleGeom(20, 20, 1),
				Start:  time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
				End:    time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			WantIDs: nil,
		},
		{
			Name: "ends before search window",
			Events: []string{`{
				"id": "1",
				"start_time": "2000-01-01T00:00:00Z",
				"end_time": "2000-01-01T01:00:00Z",
				"place": {
					"location": {
						"street": "street addr",
						"latitude": 20,
						"longitude": 20
					}
				}
			}`},
			Search: eventdb.EventSearchRequest{
				Bounds: geojson.CircleGeom(20, 20, 1),
				Start:  time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC),
				End:    time.Date(2000, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			WantIDs: nil,
		},
		{
			Name: "starts after search window",
			Events: []string{`{
				"id": "1",
				"start_time": "2001-01-01T00:00:00Z",
				"place": {
					"location": {
						"street": "street addr",
						"latitude": 20,
						"longitude": 20
					}
				}
			}`},
			Search: eventdb.EventSearchRequest{
				Bounds: geojson.CircleGeom(20, 20, 1),
				Start:  time.Date(2000, 1, 1, 1, 0, 0, 0, time.UTC),
				End:    time.Date(2000, 1, 1, 2, 0, 0, 0, time.UTC),
			},
			WantIDs: nil,
		},
		{
			Name: "starts in middle of search window",
			Events: []string{`{
				"id": "1",
				"start_time": "2000-01-01T01:00:00Z",
				"end_time": "2000-01-01T02:00:00Z",
				"place": {
					"location": {
						"street": "street addr",
						"latitude": 20,
						"longitude": 20
					}
				}
			}`},
			Search: eventdb.EventSearchRequest{
				Bounds: geojson.CircleGeom(20, 20, 1),
				Start:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				End:    time.Date(2000, 1, 1, 5, 0, 0, 0, time.UTC),
			},
			WantIDs: []eventdb.EventID{"1"},
		},
		{
			Name: "ends after search window",
			Events: []string{`{
				"id": "1",
				"start_time": "2000-01-01T01:00:00Z",
				"end_time": "2000-01-01T03:00:00Z",
				"place": {
					"location": {
						"street": "street addr",
						"latitude": 20,
						"longitude": 20
					}
				}
			}`},
			Search: eventdb.EventSearchRequest{
				Bounds: geojson.CircleGeom(20, 20, 1),
				Start:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				End:    time.Date(2000, 1, 1, 2, 0, 0, 0, time.UTC),
			},
			WantIDs: []eventdb.EventID{"1"},
		},
		{
			Name: "bad event",
			Events: []string{`{
				"id": "1",
				"start_time": "2000-01-01T00:00:00Z",
				"description": "$5 bad description",
				"place": {
					"location": {
						"street": "street addr",
						"latitude": 20,
						"longitude": 20
					}
				}
			}`},
			IsBad: true,
			Search: eventdb.EventSearchRequest{
				Bounds: geojson.CircleGeom(20, 20, 1),
				Start:  time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
				End:    time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			WantIDs: nil,
		},
		{
			Name: "include bad",
			Events: []string{`{
				"id": "1",
				"start_time": "2000-01-01T00:00:00Z",
				"description": "$5 bad description",
				"place": {
					"location": {
						"street": "street addr",
						"latitude": 20,
						"longitude": 20
					}
				}
			}`},
			IsBad: true,
			Search: eventdb.EventSearchRequest{
				Bounds:     geojson.CircleGeom(20, 20, 1),
				Start:      time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
				End:        time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
				IncludeBad: true,
			},
			WantIDs: []eventdb.EventID{"1"},
		},
	} {
		dbx := pgtest.NewDB(t)
		store := &EventStore{DB: dbx}
		if err := store.Init(ctx); err != nil {
			t.Fatal(err)
		}

		for _, e := range test.Events {
			saved, err := store.Save(ctx, json.RawMessage(e))
			if err != nil {
				t.Fatalf("event save: %v", err)
			}

			if err := store.SetBad(ctx, saved.ID, test.IsBad); err != nil {
				t.Fatalf("set bad: %v", err)
			}
		}

		res, err := store.Search(ctx, test.Search)
		if err != nil {
			t.Fatalf("event search: %v", err)
		}
		var ids []eventdb.EventID
		for _, e := range res {
			ids = append(ids, e.ID)
		}

		if got, want := ids, test.WantIDs; !reflect.DeepEqual(got, want) {
			t.Fatalf("search (%v): got ids=%v, want %v", test.Name, got, want)
		}

		fullRes, err := store.SearchFull(ctx, test.Search)
		if err != nil {
			t.Fatalf("event search (full): %v", err)
		}
		var fullIDs []eventdb.EventID
		for _, e := range fullRes {
			var fbData struct {
				ID eventdb.EventID `json:"id"`
			}
			if err := json.Unmarshal(e, &fbData); err != nil {
				t.Fatalf("SearchFull (%v): %v", test.Name, err)
			}
			fullIDs = append(fullIDs, fbData.ID)
		}

		if got, want := fullIDs, test.WantIDs; !reflect.DeepEqual(got, want) {
			t.Fatalf("search (full) (%v): got ids=%v, want %v", test.Name, got, want)
		}
	}
}

func BenchmarkSearch(b *testing.B) {
	b.Skip("this benchmark is really flaky")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbx := pgtest.NewDB(b)
	store := &EventStore{DB: dbx}
	if err := store.Init(ctx); err != nil {
		b.Fatal(err)
	}

	for i := 0; i < 500; i++ {
		id := fmt.Sprint(i)
		lat := rand.Float64() * 10
		lng := rand.Float64() * 10
		js := fmt.Sprintf(`{
				"id": %q,
				"start_time": "2000-01-01T00:00:00Z",
				"place": {
					"location": {
						"street": "street addr",
						"latitude": %f,
						"longitude": %f
					}
				}
			}`, id, lat, lng)

		if _, err := store.Save(ctx, json.RawMessage(js)); err != nil {
			b.Fatalf("save: %v", err)
		}
	}

	params := eventdb.EventSearchRequest{
		Bounds: geojson.CircleGeom(0, 0, 1),
		Start:  time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
		End:    time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, err := store.Search(ctx, params)
		if err != nil {
			b.Fatalf("search: %v", err)
		}
	}
}

func getTZ(location string) *time.Location {
	l, err := time.LoadLocation(location)
	if err != nil {
		panic(err)
	}
	return l
}
