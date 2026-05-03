package ogamex

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/user/ogame-bot/internal/model"
)

var nonNumeric = regexp.MustCompile(`[^0-9\-]`)
var versionRe = regexp.MustCompile(`(\d+\.\d+\.\d+)`)

func parseAmount(s string) int {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func parsePlanetList(body io.Reader) ([]model.Planet, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	var planets []model.Planet
	doc.Find("#planetList .smallplanet").Each(func(i int, s *goquery.Selection) {
		link := s.Find("a")
		href, exists := link.Attr("href")
		if !exists {
			return
		}

		var planetID int
		if strings.Contains(href, "cp=") {
			parts := strings.Split(href, "cp=")
			if len(parts) > 1 {
				idStr := strings.SplitN(parts[1], "&", 2)[0]
				planetID, _ = strconv.Atoi(idStr)
			}
		}
		if planetID == 0 {
			return
		}

		name := strings.TrimSpace(link.Find(".planet-name").Text())
		coordsText := strings.TrimSpace(link.Find(".planet-koords").Text())
		coordsText = strings.Trim(coordsText, "[]")

		isMoon := link.HasClass("moonlink") || strings.Contains(href, "type=moon")

		coord := parseCoordinate(coordsText)
		coordType := "planet"
		if isMoon {
			coordType = "moon"
		}
		coord.Type = coordType

		planets = append(planets, model.Planet{
			ID:         planetID,
			Name:       name,
			Coordinate: coord,
			IsMoon:     isMoon,
		})
	})

	return planets, nil
}

func parseCoordinate(s string) model.Coordinate {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return model.Coordinate{}
	}
	return model.Coordinate{
		Galaxy:   parseAmount(parts[0]),
		System:   parseAmount(parts[1]),
		Position: parseAmount(parts[2]),
	}
}

func parseResources(body io.Reader) (model.Resources, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return model.Resources{}, fmt.Errorf("parsing HTML: %w", err)
	}

	var res model.Resources
	res.Metal = parseResourceValue(doc, "resources_metal")
	res.Crystal = parseResourceValue(doc, "resources_crystal")
	res.Deuterium = parseResourceValue(doc, "resources_deuterium")
	res.Energy = parseResourceValue(doc, "resources_energy")
	res.DarkMatter = parseResourceValue(doc, "resources_darkmatter")

	return res, nil
}

func parseResourceValue(doc *goquery.Document, id string) int {
	sel := doc.Find("#" + id)
	if raw, exists := sel.Attr("data-raw"); exists {
		return parseAmount(raw)
	}
	return parseAmount(sel.Text())
}

func parseBuildings(body io.Reader, idMap map[int]*int) error {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return fmt.Errorf("parsing HTML: %w", err)
	}

	doc.Find("li.technology").Each(func(i int, s *goquery.Selection) {
		techStr, exists := s.Attr("data-technology")
		if !exists {
			return
		}
		techID, err := strconv.Atoi(techStr)
		if err != nil {
			return
		}

		ptr, ok := idMap[techID]
		if !ok {
			return
		}

		if levelStr, exists := s.Find(".level").Attr("data-level"); exists {
			*ptr = parseAmount(levelStr)
		} else {
			levelText := s.Find(".level").Text()
			*ptr = parseAmount(levelText)
		}
	})

	return nil
}

func parseAmounts(body io.Reader, idMap map[int]*int) error {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return fmt.Errorf("parsing HTML: %w", err)
	}

	doc.Find("li.technology").Each(func(i int, s *goquery.Selection) {
		techStr, exists := s.Attr("data-technology")
		if !exists {
			return
		}
		techID, err := strconv.Atoi(techStr)
		if err != nil {
			return
		}

		ptr, ok := idMap[techID]
		if !ok {
			return
		}

		if val, exists := s.Find(".amount").Attr("data-value"); exists {
			*ptr = parseAmount(val)
		} else {
			*ptr = parseAmount(s.Find(".amount").Text())
		}
	})

	return nil
}

func parseResourceBuildings(body io.Reader) (model.ResourceBuildings, error) {
	var b model.ResourceBuildings
	idMap := map[int]*int{
		1:  &b.MetalMine,
		2:  &b.CrystalMine,
		3:  &b.DeuteriumSynthesizer,
		4:  &b.SolarPlant,
		12: &b.FusionReactor,
		22: &b.MetalStorage,
		23: &b.CrystalStorage,
		24: &b.DeuteriumStorage,
	}
	if err := parseBuildings(body, idMap); err != nil {
		return model.ResourceBuildings{}, err
	}
	return b, nil
}

func parseFacilities(body io.Reader) (model.Facilities, error) {
	var f model.Facilities
	idMap := map[int]*int{
		14: &f.RoboticsFactory,
		21: &f.Shipyard,
		31: &f.ResearchLab,
		34: &f.AllianceDepot,
		44: &f.MissileSilo,
		15: &f.NaniteFactory,
		33: &f.Terraformer,
		36: &f.SpaceDock,
	}
	if err := parseBuildings(body, idMap); err != nil {
		return model.Facilities{}, err
	}
	return f, nil
}

func parseShips(body io.Reader) (model.Ships, error) {
	var s model.Ships
	idMap := map[int]*int{
		202: &s.SmallCargo,
		203: &s.LargeCargo,
		204: &s.LightFighter,
		205: &s.HeavyFighter,
		206: &s.Cruiser,
		207: &s.Battleship,
		208: &s.ColonyShip,
		209: &s.Recycler,
		210: &s.EspionageProbe,
		211: &s.Bomber,
		212: &s.SolarSatellite,
		213: &s.Destroyer,
		214: &s.Deathstar,
		215: &s.Battlecruiser,
	}
	if err := parseAmounts(body, idMap); err != nil {
		return model.Ships{}, err
	}
	return s, nil
}

func parseDefence(body io.Reader) (model.Defence, error) {
	var d model.Defence
	idMap := map[int]*int{
		401: &d.RocketLauncher,
		402: &d.LightLaser,
		403: &d.HeavyLaser,
		404: &d.GaussCannon,
		405: &d.IonCannon,
		406: &d.PlasmaTurret,
		407: &d.SmallShield,
		408: &d.LargeShield,
		502: &d.AntiBallisticMissile,
		503: &d.InterplanetaryMissile,
	}
	if err := parseAmounts(body, idMap); err != nil {
		return model.Defence{}, err
	}
	return d, nil
}

func parseResearch(body io.Reader) (model.Research, error) {
	var r model.Research
	idMap := map[int]*int{
		113: &r.EnergyTechnology,
		120: &r.LaserTechnology,
		121: &r.IonTechnology,
		114: &r.HyperspaceTechnology,
		122: &r.PlasmaTechnology,
		115: &r.CombustionDrive,
		117: &r.ImpulseDrive,
		118: &r.HyperspaceDrive,
		106: &r.EspionageTechnology,
		108: &r.ComputerTechnology,
		124: &r.Astrophysics,
		123: &r.IntergalacticResearchNetwork,
		199: &r.GravitonTechnology,
		109: &r.WeaponTechnology,
		110: &r.ShieldingTechnology,
		111: &r.ArmourTechnology,
	}
	if err := parseBuildings(body, idMap); err != nil {
		return model.Research{}, err
	}
	return r, nil
}

func parseConstructions(body io.Reader) (model.Constructions, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return model.Constructions{}, fmt.Errorf("parsing HTML: %w", err)
	}

	var c model.Constructions

	buildCountdown := doc.Find("#eventbox .countdown")
	if buildCountdown.Length() > 0 {
		text := buildCountdown.Text()
		if text != "" {
			if timeVal, exists := buildCountdown.Attr("data-time"); exists {
				c.Building.Countdown = int64(parseAmount(timeVal))
			}
		}
	}

	return c, nil
}

type eventboxResponse struct {
	Hostile    int    `json:"hostile"`
	Friendly   int    `json:"friendly"`
	Neutral    int    `json:"neutral"`
	ServerTime string `json:"serverTime"`
}

func parseEventbox(body []byte) (eventboxResponse, error) {
	var e eventboxResponse
	if err := jsonUnmarshal(body, &e); err != nil {
		return eventboxResponse{}, fmt.Errorf("parsing eventbox JSON: %w", err)
	}
	return e, nil
}

func parseFleetEvents(body []byte) ([]model.Fleet, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parsing fleet HTML: %w", err)
	}

	var fleets []model.Fleet
	doc.Find(".eventFleet").Each(func(i int, s *goquery.Selection) {
		idStr, exists := s.Attr("id")
		if !exists {
			return
		}
		idParts := strings.SplitN(idStr, "-", 2)
		fleetID := int64(0)
		if len(idParts) == 2 {
			fleetID = int64(parseAmount(idParts[1]))
		}

		missionStr, _ := s.Attr("data-mission-type")
		returnStr, _ := s.Attr("data-return-flight")

		originCoords := parseCoordinateFromBrackets(s.Find(".originCoords").Text())
		destCoords := parseCoordinateFromBrackets(s.Find(".destCoords").Text())

		var arrivalTime int64
		if timeVal, exists := s.Find(".countdown").Attr("data-time"); exists {
			arrivalTime = int64(parseAmount(timeVal))
		}

		var metal, crystal, deuterium int
		details := strings.TrimSpace(s.Find(".details").Text())
		if details != "" {
			parts := strings.Split(details, ",")
			if len(parts) >= 1 {
				metal = parseAmount(parts[0])
			}
			if len(parts) >= 2 {
				crystal = parseAmount(parts[1])
			}
			if len(parts) >= 3 {
				deuterium = parseAmount(parts[2])
			}
		}

		fleets = append(fleets, model.Fleet{
			ID:           int(fleetID),
			Mission:      parseAmount(missionStr),
			ReturnFlight: returnStr == "true",
			Origin:       originCoords,
			Destination:  destCoords,
			ArrivalTime:  arrivalTime,
			Metal:        metal,
			Crystal:      crystal,
			Deuterium:    deuterium,
		})
	})

	return fleets, nil
}

func parseCoordinateFromBrackets(s string) model.Coordinate {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "[]")
	return parseCoordinate(s)
}

func parseSlots(body io.Reader) (model.Slots, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return model.Slots{}, fmt.Errorf("parsing HTML: %w", err)
	}

	var slots model.Slots
	doc.Find("#slots .fleetSlots").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		parts := strings.Split(text, "/")
		if len(parts) == 2 {
			slots.InUse = parseAmount(parts[0])
			slots.Total = parseAmount(parts[1])
		}
	})

	doc.Find("#slots .expSlots").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		parts := strings.Split(text, "/")
		if len(parts) == 2 {
			slots.ExpInUse = parseAmount(parts[0])
			slots.ExpTotal = parseAmount(parts[1])
		}
	})

	return slots, nil
}

func parseServerInfo(body []byte) (speed int, version string) {
	speed = 1
	version = "unknown"

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return speed, version
	}

	doc.Find("#footer span, #footer").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if m := versionRe.FindString(text); m != "" {
			version = m
		}
	})

	factorEl := doc.Find("#overviewBottom .factor")
	if val, exists := factorEl.Attr("data-value"); exists {
		speed = parseAmount(val)
	}

	return speed, version
}
