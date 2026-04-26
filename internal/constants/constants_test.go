package constants

import "testing"

func TestMissionConstants(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  int
	}{
		{"MissionAttack", MissionAttack, 1},
		{"MissionUnionAttack", MissionUnionAttack, 2},
		{"MissionDeploy", MissionDeploy, 3},
		{"MissionTransport", MissionTransport, 4},
		{"MissionUnionTransport", MissionUnionTransport, 5},
		{"MissionRelocate", MissionRelocate, 6},
		{"MissionStation", MissionStation, 7},
		{"MissionEspionage", MissionEspionage, 6},
		{"MissionColonize", MissionColonize, 7},
		{"MissionHarvest", MissionHarvest, 8},
		{"MissionExpedition", MissionExpedition, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
			}
		})
	}
}

func TestBuildingConstants(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  int
	}{
		{"BuildingMetalMine", BuildingMetalMine, 1},
		{"BuildingCrystalMine", BuildingCrystalMine, 2},
		{"BuildingDeuteriumSynthesizer", BuildingDeuteriumSynthesizer, 3},
		{"BuildingSolarPlant", BuildingSolarPlant, 4},
		{"BuildingFusionReactor", BuildingFusionReactor, 12},
		{"BuildingMetalStorage", BuildingMetalStorage, 22},
		{"BuildingCrystalStorage", BuildingCrystalStorage, 23},
		{"BuildingDeuteriumStorage", BuildingDeuteriumStorage, 24},
		{"BuildingRoboticsFactory", BuildingRoboticsFactory, 14},
		{"BuildingShipyard", BuildingShipyard, 21},
		{"BuildingResearchLab", BuildingResearchLab, 31},
		{"BuildingAllianceDepot", BuildingAllianceDepot, 34},
		{"BuildingMissileSilo", BuildingMissileSilo, 44},
		{"BuildingNaniteFactory", BuildingNaniteFactory, 15},
		{"BuildingTerraformer", BuildingTerraformer, 33},
		{"BuildingSpaceDock", BuildingSpaceDock, 36},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
			}
		})
	}
}

func TestShipConstants(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  int
	}{
		{"ShipSmallCargo", ShipSmallCargo, 202},
		{"ShipLargeCargo", ShipLargeCargo, 203},
		{"ShipLightFighter", ShipLightFighter, 204},
		{"ShipHeavyFighter", ShipHeavyFighter, 205},
		{"ShipCruiser", ShipCruiser, 206},
		{"ShipBattleship", ShipBattleship, 207},
		{"ShipColonyShip", ShipColonyShip, 208},
		{"ShipRecycler", ShipRecycler, 209},
		{"ShipEspionageProbe", ShipEspionageProbe, 210},
		{"ShipBomber", ShipBomber, 211},
		{"ShipSolarSatellite", ShipSolarSatellite, 212},
		{"ShipDestroyer", ShipDestroyer, 213},
		{"ShipDeathstar", ShipDeathstar, 214},
		{"ShipBattlecruiser", ShipBattlecruiser, 215},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
			}
		})
	}
}
