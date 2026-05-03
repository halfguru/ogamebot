package ogamex

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/user/ogame-bot/internal/model"
)

func (c *Client) GetEspionageReportMessages(ctx context.Context) ([]model.EspionageReportSummary, error) {
	body, err := c.doAJAXGet(ctx, "/ajax/messages?tab=fleets&subtab=espionage")
	if err != nil {
		return nil, fmt.Errorf("fetching espionage messages: %w", err)
	}
	return parseEspionageMessageList(body)
}

func (c *Client) GetEspionageReport(ctx context.Context, messageID int64) (model.EspionageReport, error) {
	path := fmt.Sprintf("/ajax/messages/%d?tab=fleets&subtab=espionage", messageID)
	body, err := c.doAJAXGet(ctx, path)
	if err != nil {
		return model.EspionageReport{}, fmt.Errorf("fetching espionage report %d: %w", messageID, err)
	}
	return parseEspionageReport(body)
}

func (c *Client) DeleteAllEspionageReports(ctx context.Context) error {
	msgs, err := c.GetEspionageReportMessages(ctx)
	if err != nil {
		return fmt.Errorf("listing reports for deletion: %w", err)
	}
	for _, msg := range msgs {
		data := url.Values{}
		data.Set("action", "103")
		data.Set("messageId", strconv.FormatInt(msg.ID, 10))
		_, err := c.doPost(ctx, "/messages", data)
		if err != nil {
			return fmt.Errorf("deleting espionage report %d: %w", msg.ID, err)
		}
	}
	return nil
}

func parseEspionageMessageList(body []byte) ([]model.EspionageReportSummary, error) {
	doc, err := goquery.NewDocumentFromReader(toReader(body))
	if err != nil {
		return nil, fmt.Errorf("parsing espionage message list HTML: %w", err)
	}

	var reports []model.EspionageReportSummary
	doc.Find(".message").Each(func(i int, s *goquery.Selection) {
		link := s.Find("a")
		href, exists := link.Attr("href")
		if !exists {
			return
		}

		var messageID int64
		if idx := strings.Index(href, "/messages/"); idx != -1 {
			idPart := href[idx+len("/messages/"):]
			idPart = strings.SplitN(idPart, "?", 2)[0]
			idPart = strings.SplitN(idPart, "/", 2)[0]
			messageID, _ = strconv.ParseInt(idPart, 10, 64)
		}
		if messageID == 0 {
			if idStr, exists := s.Attr("data-message-id"); exists {
				messageID, _ = strconv.ParseInt(idStr, 10, 64)
			}
		}
		if messageID == 0 {
			return
		}

		text := link.Text()
		coordStr := extractCoordinateString(text)
		coord := parseCoordinate(coordStr)

		var dateStr string
		dateEl := s.Find(".date, .message-date, time")
		if dateEl.Length() > 0 {
			dateStr = strings.TrimSpace(dateEl.First().Text())
		}

		reports = append(reports, model.EspionageReportSummary{
			ID:         messageID,
			Coordinate: coord,
			Date:       parseDate(dateStr),
		})
	})

	return reports, nil
}

func parseEspionageReport(body []byte) (model.EspionageReport, error) {
	doc, err := goquery.NewDocumentFromReader(toReader(body))
	if err != nil {
		return model.EspionageReport{}, fmt.Errorf("parsing espionage report HTML: %w", err)
	}

	var report model.EspionageReport

	idStr, exists := doc.Find("[data-message-id]").Attr("data-message-id")
	if exists {
		report.ID, _ = strconv.ParseInt(idStr, 10, 64)
	}

	coordStr := extractCoordinateString(doc.Find(".coords, .espionageCoords, .report-target").Text())
	report.Coordinate = parseCoordinate(coordStr)

	report.Metal = int64(parseResourceValue(doc, "resources_metal"))
	report.Crystal = int64(parseResourceValue(doc, "resources_crystal"))
	report.Deuterium = int64(parseResourceValue(doc, "resources_deuterium"))

	defSection := doc.Find(".defense, .espionage-defense, #defense")
	report.HasDefensesInformation = defSection.Length() > 0

	fleetSection := doc.Find(".fleet, .espionage-fleet, #fleet")
	report.HasFleetInformation = fleetSection.Length() > 0

	if report.HasDefensesInformation {
		report.RocketLauncher = parseDefenseCount(defSection, "401")
		report.LightLaser = parseDefenseCount(defSection, "402")
		report.HeavyLaser = parseDefenseCount(defSection, "403")
		report.GaussCannon = parseDefenseCount(defSection, "404")
		report.IonCannon = parseDefenseCount(defSection, "405")
		report.PlasmaTurret = parseDefenseCount(defSection, "406")
		report.SmallShieldDome = parseDefenseCount(defSection, "407")
		report.LargeShieldDome = parseDefenseCount(defSection, "408")
	}

	return report, nil
}

func extractCoordinateString(text string) string {
	start := strings.Index(text, "[")
	end := strings.Index(text, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return text[start+1 : end]
}

func parseDefenseCount(sel *goquery.Selection, techID string) int {
	el := sel.Find(fmt.Sprintf("[data-technology='%s'] .amount, [data-id='%s'] .amount", techID, techID))
	if el.Length() > 0 {
		if val, exists := el.Attr("data-value"); exists {
			return parseAmount(val)
		}
		return parseAmount(el.Text())
	}
	el = sel.Find(fmt.Sprintf("[data-technology='%s'] .count, [data-id='%s'] .count", techID, techID))
	if el.Length() > 0 {
		return parseAmount(el.Text())
	}
	return 0
}

func parseDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		"02.01.2006 15:04:05",
		"2006-01-02 15:04:05",
		"01/02/2006 15:04:05",
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
