export const MISSION_TYPE = {
  ATTACK: 1,
  UNION_ATTACK: 2,
  DEPLOY: 3,
  TRANSPORT: 4,
  UNION_TRANSPORT: 5,
  RELOCATE: 6,
  STATION: 7,
  ESPIONAGE: 6,
  COLONIZE: 7,
  HARVEST: 8,
  EXPEDITION: 15,
} as const;

export type MissionType = (typeof MISSION_TYPE)[keyof typeof MISSION_TYPE];
