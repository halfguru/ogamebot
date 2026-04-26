export interface Coordinate {
  galaxy: number;
  system: number;
  position: number;
  type: 'planet' | 'moon';
}

export interface Resources {
  metal: number;
  crystal: number;
  deuterium: number;
  energy: number;
  darkmatter?: number;
}

export interface Planet {
  id: number;
  name: string;
  coordinate: Coordinate;
  diameter: number;
  fieldsUsed: number;
  fieldsTotal: number;
  temperatureMin: number;
  temperatureMax: number;
  isMoon: boolean;
}
