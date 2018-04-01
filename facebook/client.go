package facebook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/findrandomevents/eventdb/log"
	"go.uber.org/zap"
)

const apiVersion = "v2.9"

// Client is a slimmed-down Facebook Graph API client.
type Client struct {
	HTTP *http.Client
}

// GetEventInfo fetches information for up to 50 Facebook event IDs using the
// Facebook Graph API. If some events do not exist or are inaccessible this
// function may return fewer event infos than the number of ids passed in.
func (f *Client) GetEventInfo(ctx context.Context, ids []string) ([]json.RawMessage, error) {
	logger := log.FromContext(ctx)

	const fields = `attending_count,can_guests_invite,can_viewer_post,category,cover,declined_count,description,end_time,guest_list_enabled,interested_count,is_canceled,is_draft,is_page_owned,is_viewer_admin,id,maybe_count,name,noreply_count,owner,parent_group,place,start_time,ticket_uri,timezone,type,updated_time`

	reqs := make([]map[string]string, len(ids))
	for i, id := range ids {
		reqs[i] = map[string]string{
			"method":       "GET",
			"relative_url": fmt.Sprintf("%s/%s?fields=%s", apiVersion, id, fields),
		}
	}
	req := map[string]interface{}{"batch": reqs}

	batchBody := bytes.NewBuffer(nil)
	if err := json.NewEncoder(batchBody).Encode(req); err != nil {
		return nil, err
	}

	resp, err := f.HTTP.Post("https://graph.facebook.com", "application/json", batchBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp.Body)
	}

	var responses []BatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&responses); err != nil {
		return nil, err
	}

	var events []json.RawMessage
	for i, r := range responses {
		if r.Code != http.StatusOK {
			fbErr := parseError(strings.NewReader(r.Body))

			// TODO(maxhawkins): ignore deleted event response
			logger.Error("bad event fetch response",
				zap.String("eventID", ids[i]),
				zap.Int("code", fbErr.Code),
				zap.Int("subcode", fbErr.Subcode),
				zap.String("error", fbErr.Message),
				zap.String("errorType", fbErr.Type))

			return events, fbErr
		}
		events = append(events, json.RawMessage(r.Body))
	}

	return events, nil
}
