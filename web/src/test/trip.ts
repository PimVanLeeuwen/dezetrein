import type { Stop, Trip } from '../api/types';

// IC 2450 Lelystad to Rotterdam as recorded on 2026-11-04 (backend/testdata/ns), shortened:
// 13 to 16 minutes late, platform change at Schiphol, a passing station, two trainsets.
const ev = (planned: string, actual: string, delaySeconds: number, plannedTrack: string, actualTrack: string) => ({
  planned: `2026-11-04T${planned}:00+01:00`,
  actual: `2026-11-04T${actual}+01:00`,
  delaySeconds,
  plannedTrack,
  actualTrack,
  cancelled: false,
});

const stop = (code: string, name: string, status: Stop['status'], extra: Partial<Stop> = {}): Stop => ({ code, name, uic: code, lat: 52, lng: 5, status, ...extra });

export const tripFixture: Trip = {
  train: '2450',
  category: 'ICD',
  categoryName: 'Intercity direct',
  destination: 'Rotterdam Centraal',
  stops: [
    stop('LLS', 'Lelystad Centrum', 'ORIGIN', { departure: ev('14:53', '15:06:35', 815, '1', '1'), crowd: 'LOW', carriages: ['3304', '3108'] }),
    stop('ALMB', 'Almere Buiten', 'STOP', { arrival: ev('15:05', '15:17:41', 761, '1', '1'), departure: ev('15:05', '15:17:41', 761, '1', '1'), crowd: 'LOW' }),
    stop('ALMO', 'Almere Oostvaarders', 'PASSING'),
    stop('ASDZ', 'Amsterdam Zuid', 'STOP', { arrival: ev('15:33', '15:48:16', 916, '4', '4'), departure: ev('15:34', '15:49:54', 954, '4', '4'), crowd: 'MEDIUM' }),
    stop('SHL', 'Schiphol Airport', 'STOP', { arrival: ev('15:40', '15:56:13', 973, '6', '6'), departure: ev('15:42', '15:57:53', 953, '5-6', '6'), crowd: 'LOW' }),
    stop('RTD', 'Rotterdam Centraal', 'DESTINATION', { arrival: ev('16:08', '16:23:48', 948, '11', '11') }),
  ],
  composition: { train: '2450', type: 'ICNG', carriages: ['3304', '3108'], lengthM: 161 },
  disruptions: [{ id: 'd1', type: 'DISRUPTION', title: 'Rotterdam - Breda.', description: 'Door een wisselstoring rijden er geen treinen op de HSL.' }],
  position: null,
  as_of: '2026-11-04T14:52:00Z',
  stale: false,
};

// Moments on that trip (UTC): before departure, between Amsterdam Zuid and Schiphol, after arrival.
export const at = {
  before: Date.parse('2026-11-04T13:50:00Z'),
  betweenZuidAndSchiphol: Date.parse('2026-11-04T14:52:00Z'),
  arrived: Date.parse('2026-11-04T15:30:00Z'),
};
