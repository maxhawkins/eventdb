package pg

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/findrandomevents/eventdb"
	"github.com/findrandomevents/eventdb/errors"

	"github.com/lib/pq"
)

// EventStore stores and retrives Events from a PostgreSQL database. Events are
// stored as raw Graph API responses in a Postgres JSON database.
type EventStore struct {
	DB *sql.DB
}

// Init sets up the database schema and creates indices.
func (e *EventStore) Init(ctx context.Context) error {
	const op errors.Op = "EventStore.Init"

	_, err := e.DB.ExecContext(ctx, `
	CREATE EXTENSION IF NOT EXISTS postgis;

	-- Create a timestamptz from a text timestamp
	--
	-- NOTE(maxhawkins): this function assumes that the timestamp is
	-- in a format that's not changed by the DateStyle parameter.
	-- See: https://www.postgresql.org/docs/9.5/static/datatype-datetime.html
	CREATE OR REPLACE FUNCTION f_immutable_timestamptz(text)
	RETURNS timestamptz AS $$
		SELECT CAST($1 AS timestamptz)
	$$
	LANGUAGE sql
	IMMUTABLE;

	CREATE OR REPLACE FUNCTION f_event_start_time(jsonb)
	RETURNS timestamptz AS $$
		SELECT f_immutable_timestamptz($1->>'start_time')
	$$
	LANGUAGE sql
	IMMUTABLE;

	CREATE OR REPLACE FUNCTION f_event_end_time(jsonb)
	RETURNS timestamptz AS $$
		SELECT COALESCE(
			f_immutable_timestamptz($1->>'end_time'),
			f_event_start_time($1) + interval '1 hour'
		);
	$$
	LANGUAGE sql
	IMMUTABLE;

	CREATE OR REPLACE FUNCTION f_event_address(jsonb)
	RETURNS text AS $$
		SELECT $1->'place'->'location'->>'street'
	$$
	LANGUAGE sql
	IMMUTABLE;

	-- Extract the event's duration as a time interval
	CREATE OR REPLACE FUNCTION f_event_duration(jsonb)
	RETURNS interval AS $$
		SELECT f_event_end_time($1) - f_event_start_time($1)
	$$
	LANGUAGE sql
	IMMUTABLE;

	CREATE TABLE IF NOT EXISTS events (
     id    VARCHAR(40)   NOT NULL,
	   data  jsonb         NOT NULL,
	   is_bad   boolean,
	   geom  geometry
	);

	CREATE UNIQUE INDEX IF NOT EXISTS event_id_idx ON events (id);

	-- Geospatial index to speed up EventStore.Search
	CREATE INDEX IF NOT EXISTS event_search_idx
	ON events
	USING GIST (
		geom,
		tstzrange(f_event_start_time(data), f_event_end_time(data))
	)
	WHERE f_event_duration(data) < interval '10 hours'
	AND f_event_address(data) IS NOT NULL;
	`)
	if err != nil {
		return errors.E(op, pgErr(err))
	}

	return nil
}

// doSearch executes a search query with EventSearchRequest and returns all the
// event IDs that match.
func (e *EventStore) doSearch(ctx context.Context, params eventdb.EventSearchRequest) ([]eventdb.EventID, error) {
	rows, err := e.DB.QueryContext(ctx, `
		SELECT data->>'id' AS id
		FROM events
		WHERE
			-- Restrict to events within the given GeoJSON bounds
			ST_Within(
				geom,
				ST_CollectionExtract(
					ST_MakeValid(ST_SetSRID(ST_GeomFromGeoJSON($1), 4326)),
					3
				)
			)

			-- Events without an address are usually not specific to one place in a city
			-- and we can't draw a dot on the map
			AND f_event_address(data) IS NOT NULL

			-- Filter to events that are in the requested time window
			AND tstzrange(f_event_start_time(data), f_event_end_time(data)) && tstzrange($2, $3)

			-- Remove day-long events (not practical to attend)
			AND f_event_duration(data) < interval '10 hours'

			-- Filter out "bad" events determined uninteresting
			-- by event text analysis
			AND ($4 OR is_bad IS NULL OR is_bad = FALSE)
		`,
		params.Bounds,
		params.Start,
		params.End,
		params.IncludeBad)
	if err != nil {
		return nil, pgErr(err)
	}
	defer rows.Close()

	var eventIDs []eventdb.EventID
	for rows.Next() {
		var id eventdb.EventID
		if err = rows.Scan(&id); err != nil {
			return nil, pgErr(err)
		}
		eventIDs = append(eventIDs, id)
	}
	if err = rows.Err(); err != nil {
		return nil, pgErr(err)
	}

	return eventIDs, err
}

// Search executes a search query with EventSearchRequest and returns all the
// Events that match, with the description truncated in the database to save
// bandiwdth.
func (e *EventStore) Search(ctx context.Context, params eventdb.EventSearchRequest) ([]eventdb.Event, error) {
	eventIDs, err := e.doSearch(ctx, params)
	if err != nil {
		return nil, err
	}
	events, err := e.fetchEvents(ctx, eventIDs)
	if err != nil {
		return nil, err
	}

	return events, nil
}

// SearchFull executes a search query with EventSearchRequest and returns the raw Graph API
// JSON for all the events that match.
func (e *EventStore) SearchFull(ctx context.Context, params eventdb.EventSearchRequest) ([]json.RawMessage, error) {
	eventIDs, err := e.doSearch(ctx, params)
	if err != nil {
		return nil, err
	}
	return e.fetchEventsFull(ctx, eventIDs)
}

// Save creates or updates an Event in the database, given a JSON message from
// the Graph API.
func (e *EventStore) Save(ctx context.Context, eventJS json.RawMessage) (eventdb.Event, error) {
	var evtID struct {
		ID eventdb.EventID `json:"id"`
	}
	if err := json.Unmarshal([]byte(eventJS), &evtID); err != nil {
		return eventdb.Event{}, err
	}
	eventID := evtID.ID

	tx, err := e.DB.BeginTx(ctx, nil)
	if err != nil {
		return eventdb.Event{}, pgErr(err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO events
			(id, data)
		VALUES
			($1, $2)
		ON CONFLICT (id) DO UPDATE
			SET data=$2
		`, eventID, []byte(eventJS))
	if err != nil {
		return eventdb.Event{}, errors.E(pgErr(err), "insert event")
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE events
		SET geom = ST_SetSRID(ST_MakePoint(
			(data->'place'->'location'->>'longitude')::float,
			(data->'place'->'location'->>'latitude')::float), 4326)
		WHERE
			id = $1
	`, eventID)
	if err != nil {
		return eventdb.Event{}, errors.E(pgErr(err), "set geom")
	}

	if err = tx.Commit(); err != nil {
		return eventdb.Event{}, pgErr(err)
	}

	event, err := e.GetByID(ctx, eventID)
	if err != nil {
		return event, err
	}

	return event, nil
}

// SetBad updates an event's 'bad' flag, which determines whether it gets
// filtered from search results.
func (e *EventStore) SetBad(ctx context.Context, eventID eventdb.EventID, isBad bool) error {
	_, err := e.DB.ExecContext(ctx, `
	UPDATE events
	SET is_bad = $1
	WHERE id = $2
	`, isBad, eventID)
	if err != nil {
		return err
	}

	return nil
}

// GetByID finds an event by its ID
func (e *EventStore) GetByID(ctx context.Context, eventID eventdb.EventID) (eventdb.Event, error) {
	events, err := e.fetchEvents(ctx, []eventdb.EventID{eventID})
	if err != nil {
		return eventdb.Event{}, errors.E(err)
	}

	if len(events) == 0 {
		return eventdb.Event{}, errors.E(errors.NotExist)
	}

	event := events[0]
	return event, nil
}

// GetMulti finds multiple events simultaneously by their IDs.
func (e *EventStore) GetMulti(ctx context.Context, eventIDs []eventdb.EventID) ([]eventdb.Event, error) {
	events, err := e.fetchEvents(ctx, eventIDs)
	if err != nil {
		return events, errors.E(err, "get multi")
	}

	return events, nil
}

func (e *EventStore) fetchEvents(ctx context.Context, eventIDs []eventdb.EventID) ([]eventdb.Event, error) {
	events := []eventdb.Event{}

	var idStrings pq.StringArray
	for _, id := range eventIDs {
		idStrings = append(idStrings, string(id))
	}

	rows, err := e.DB.QueryContext(ctx, `
	SELECT
		COALESCE(data->>'id', '') AS id,

		COALESCE(data->>'name', '') AS name,
		COALESCE(data->'cover'->>'source', '') AS cover,
		f_event_start_time(data) AS start_time,
		f_event_end_time(data) AS end_time,
		COALESCE( ST_Y(ST_Transform(geom, 4326)), 0) AS latitude,
		COALESCE( ST_X(ST_Transform(geom, 4326)), 0) AS longitude,

		COALESCE(data->>'is_canceled', 'false') AS is_canceled,

		COALESCE(is_bad, 'false'),

        COALESCE(data->>'description', '') AS description,

		COALESCE(data->'place'->>'name', '') AS place,
		COALESCE(f_event_address(data), '') AS address,

		COALESCE(data->>'timezone', '') AS timezone

	FROM events
	WHERE
		id = ANY ($1)
	ORDER BY start_time ASC
	`, idStrings)
	if err != nil {
		return events, errors.E(pgErr(err), "select events")
	}
	defer rows.Close()

	for rows.Next() {
		var timezone string

		var event eventdb.Event
		err = rows.Scan(
			&event.ID,
			&event.Name,
			&event.Cover,
			&event.StartTime,
			&event.EndTime,
			&event.Latitude,
			&event.Longitude,
			&event.IsCanceled,
			&event.IsBad,
			&event.Description,
			&event.Place,
			&event.Address,
			&timezone,
		)
		if err != nil {
			return events, err
		}

		location, err := time.LoadLocation(timezone)
		if err != nil {
			location = time.UTC
		}

		event.StartTime = event.StartTime.In(location)
		event.EndTime = event.EndTime.In(location)

		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return events, err
	}

	return events, nil
}

func (e *EventStore) fetchEventsFull(ctx context.Context, eventIDs []eventdb.EventID) ([]json.RawMessage, error) {
	events := []json.RawMessage{}

	var idStrings pq.StringArray
	for _, id := range eventIDs {
		idStrings = append(idStrings, string(id))
	}

	rows, err := e.DB.QueryContext(ctx, `
	SELECT
		data::text AS data
	FROM events
	WHERE
		id = ANY ($1)
	ORDER BY f_event_start_time(data) ASC
	`, idStrings)
	if err != nil {
		return events, errors.E(pgErr(err), "select events")
	}
	defer rows.Close()

	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, pgErr(err)
		}

		var m json.RawMessage
		if err := json.Unmarshal(data, &m); err != nil {
			return events, err
		}
		events = append(events, m)
	}
	if err := rows.Err(); err != nil {
		return nil, pgErr(err)
	}

	return events, nil
}
