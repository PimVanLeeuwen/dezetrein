// The backend's JSON, as served by backend/internal/api. Times are RFC 3339 strings.

export type StopStatus = 'ORIGIN' | 'STOP' | 'PASSING' | 'DESTINATION';
export type Crowd = 'LOW' | 'MEDIUM' | 'HIGH' | 'UNKNOWN';

export type TripEvent = {
  planned: string;
  actual?: string;
  delaySeconds: number;
  plannedTrack?: string;
  actualTrack?: string;
  cancelled: boolean;
};

export type Stop = {
  code: string;
  name: string;
  uic: string;
  lat: number;
  lng: number;
  status: StopStatus;
  arrival?: TripEvent;
  departure?: TripEvent;
  crowd?: Crowd | string;
  carriages?: string[];
};

export type Composition = { train: string; type: string; carriages: string[]; lengthM?: number };

export type Disruption = {
  id: string;
  type: 'DISRUPTION' | 'MAINTENANCE' | 'CALAMITY' | string;
  title: string;
  description?: string;
  start?: string;
  end?: string;
  impact?: number;
  stations?: string[];
  url?: string;
};

export type Position = { lat: number; lng: number; speedKmh: number; headingDeg: number; as_of: string };

export type Trip = {
  train: string;
  category: string;
  categoryName: string;
  destination: string;
  stops: Stop[];
  composition: Composition | null;
  disruptions: Disruption[];
  position: Position | null;
  as_of: string;
  stale: boolean;
};

export type Candidate = {
  train: string;
  category?: string;
  destination?: string;
  carriages: string[];
  distanceM?: number;
};

export type Confidence = 'high' | 'low' | 'none';

export type MatchResult = {
  match: string | null;
  confidence: Confidence;
  candidates: Candidate[];
  as_of: string;
  stale: boolean;
};

export type CarriageResult = {
  carriage: string;
  train: Candidate | null;
  source?: 'index' | 'nearby';
  located: boolean;
  searched: number;
  as_of?: string;
  stale: boolean;
};

export type Departure = {
  train: string;
  category: string;
  direction: string;
  planned: string;
  actual?: string;
  plannedTrack?: string;
  actualTrack?: string;
  cancelled: boolean;
  messages?: string[];
};

export type Departures = { station: string; departures: Departure[]; as_of: string; stale: boolean };

// A location reading as the backend accepts it: speed and heading are null when the phone does
// not report them; t is the Geolocation API timestamp in milliseconds.
export type Fix = { lat: number; lng: number; accuracy: number; speed: number | null; heading: number | null; t: number };

export type NearbyStation = { code: string; name: string; distanceM: number };

export type NearestStations = { stations: NearbyStation[]; as_of: string; stale: boolean };
