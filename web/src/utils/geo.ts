export type LatLngTuple = [number, number]

export function normalizeRoute(
  osrmCoords: [number, number][]
): LatLngTuple[] {
  return osrmCoords.map(([lng, lat]) => [lat, lng])
}
