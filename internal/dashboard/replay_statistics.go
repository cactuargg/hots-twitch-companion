package dashboard

import (
	"errors"
	"hots-twitch-companion/internal/hots/correlation"
	"hots-twitch-companion/internal/hots/discovery"
	"hots-twitch-companion/internal/hots/lobby"
	"hots-twitch-companion/internal/hots/replay"
	"hots-twitch-companion/internal/replaystats"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func parseStatisticsRecord(raw []byte, now time.Time) (replaystats.Record, error) {
	var m replaystats.Record
	m.ID = correlation.ContentSHA(raw)
	member, err := replay.ReadMPQMember(raw, "replay.details")
	if err != nil {
		return m, err
	}
	m.Replay, err = replay.ParseDetails(member)
	if err != nil {
		return m, err
	}
	if m.Replay.RecordedAt <= 0 || m.Replay.RecordedAt > now.Add(2*time.Minute).UnixMilli() {
		return m, errors.New("replay has no valid game timestamp")
	}
	if b, e := replay.ReadMPQMember(raw, "replay.server.battlelobby"); e == nil {
		m.LobbyHash = correlation.ContentSHA(b)
		if l, e := lobby.Parse(b); e == nil {
			m.Mode = l.AMMID
		}
	}
	return m, nil
}
func (d *Dashboard) readStatisticsRecord(f discovery.ReplayFile, now time.Time, pending map[string]replaystats.Record) (replaystats.Record, error) {
	if now.Sub(f.Modified) < 8*time.Second {
		return replaystats.Record{}, errors.New("replay is still being written; import it later")
	}
	raw, err := correlation.ReadStableReplay(f)
	if err != nil {
		return replaystats.Record{}, err
	}
	id := correlation.ContentSHA(raw)
	if m, ok := pending[id]; ok {
		return m, nil
	}
	if m, ok := d.statistics.Get(id); ok {
		return m, nil
	}
	return parseStatisticsRecord(raw, now)
}
func (d *Dashboard) statisticsAPI(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "all"
	}
	if mode != "all" {
		n, err := strconv.ParseUint(mode, 10, 32)
		if err != nil {
			RequestError(w, errors.New("invalid statistics mode"))
			return
		}
		mode = strconv.FormatUint(n, 10)
		if mode != "0" && mode != "50001" && mode != "50051" && mode != "50091" && mode != "50101" {
			RequestError(w, errors.New("unsupported statistics mode"))
			return
		}
	}
	player := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("player")))
	result, err := d.statistics.Summary(mode, player)
	if err != nil {
		RequestError(w, err)
		return
	}
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "heroes"
	}
	if kind != "heroes" && kind != "players" {
		RequestError(w, errors.New("invalid statistics kind"))
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	all := result.Heroes
	if kind == "players" {
		all = result.Players
	}
	search := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("search")))
	filtered := make([]replaystats.Row, 0, len(all))
	for _, row := range all {
		if search == "" || strings.Contains(strings.ToLower(row.Name), search) || strings.Contains(row.Key, search) {
			filtered = append(filtered, row)
		}
	}
	const size = 25
	pages := (len(filtered) + size - 1) / size
	if pages == 0 {
		pages = 1
	}
	if page > pages {
		page = pages
	}
	first := (page - 1) * size
	last := first + size
	if last > len(filtered) {
		last = len(filtered)
	}
	JSONOutput(w, 200, map[string]any{"mode": mode, "kind": kind, "player": player, "matches": result.Matches, "observations": result.Observations, "skipped": result.Skipped, "from": result.From, "to": result.To, "total": len(filtered), "page": page, "pages": pages, "rows": filtered[first:last]})
}
