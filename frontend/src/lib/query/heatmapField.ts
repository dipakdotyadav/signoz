import type {
	IBuilderQuery,
	OrderByPayload,
} from 'types/api/queryBuilder/queryBuilderData';
import { DataSource } from 'types/common/queryBuilder';

import { HEATMAP_FIELD } from './heatmapFieldPattern';

// A numeric one-argument call such as `p99(duration_nano)`; the argument is the field.
// Other functions (`count_distinct`, `countIf`) take text or a condition.
const NUMERIC_FIELD_CALL =
	/^\s*(?:avg|sum|min|max|p\d{2})\(\s*([^\s(),]+)\s*\)\s*$/;

// Same default the histogram-metric switch-back picks.
const FIELD_FUNCTION = 'p90';

type Aggregation = { expression?: string; alias?: string };

function isLogsOrTraces(
	dataSource: DataSource,
): dataSource is DataSource.LOGS | DataSource.TRACES {
	return dataSource === DataSource.LOGS || dataSource === DataSource.TRACES;
}

/** Order-by names an aggregation by its alias when it has one, else its expression. */
function withExpression(
	query: IBuilderQuery,
	from: Aggregation | undefined,
	to: string,
): IBuilderQuery {
	const fromKey = from?.alias || from?.expression;
	return {
		...query,
		aggregations: [{ expression: to }] as IBuilderQuery['aggregations'],
		orderBy: query.orderBy?.map((order): OrderByPayload =>
			fromKey && order.columnName === fromKey
				? { ...order, columnName: to }
				: order,
		),
	};
}

export function getHeatmapFieldPlaceholder(dataSource: DataSource): string {
	return dataSource === DataSource.TRACES
		? 'Numeric field to bucket, e.g. duration_nano'
		: 'Numeric field to bucket';
}

/**
 * Logs and traces heatmaps bucket a bare field; any function aggregation is
 * rejected, so a numeric `fn(field)` keeps its field and anything else leaves
 * the field for the user to pick.
 */
export function withHeatmapField(query: IBuilderQuery): IBuilderQuery {
	if (!isLogsOrTraces(query.dataSource)) {
		return query;
	}
	const aggregations = query.aggregations as Aggregation[] | undefined;
	const [first] = aggregations ?? [];
	const single = aggregations?.length === 1;
	if (single && HEATMAP_FIELD.test(first.expression ?? '')) {
		return query;
	}
	const field =
		(single && NUMERIC_FIELD_CALL.exec(first.expression ?? '')?.[1]) || '';
	return withExpression(query, first, field);
}

/** Inverse of `withHeatmapField`: every other panel needs a function aggregation. */
export function withoutHeatmapField(query: IBuilderQuery): IBuilderQuery {
	if (!isLogsOrTraces(query.dataSource) || query.aggregations?.length !== 1) {
		return query;
	}
	const [first] = query.aggregations as Aggregation[];
	const expression = first.expression?.trim() ?? '';
	const field = HEATMAP_FIELD.exec(expression)?.[1];
	if (expression && !field) {
		return query;
	}
	return withExpression(
		query,
		{ alias: first.alias, expression: field },
		field ? `${FIELD_FUNCTION}(${field})` : 'count()',
	);
}
