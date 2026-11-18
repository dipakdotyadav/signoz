/**
 * A logs or traces heatmap's aggregation: one field with an optional alias.
 * The field mirrors the querier's HeatmapAggregationField: no call, no whitespace.
 */
export const HEATMAP_FIELD =
	/^\s*([^\s()]+)(?:\s+as\s+((?:'[^']*'|"[^"]*"|[a-zA-Z0-9_-]+)))?\s*$/;
