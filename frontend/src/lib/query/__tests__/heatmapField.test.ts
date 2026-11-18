import { initialQueriesMap } from 'constants/queryBuilder';
import type { IBuilderQuery } from 'types/api/queryBuilder/queryBuilderData';
import { DataSource } from 'types/common/queryBuilder';

import {
	getHeatmapFieldPlaceholder,
	withHeatmapField,
	withoutHeatmapField,
} from '../heatmapField';

const queryOf = (
	dataSource: DataSource,
	expressions: string[],
): IBuilderQuery => ({
	...(initialQueriesMap[dataSource].builder.queryData[0] as IBuilderQuery),
	aggregations: expressions.map((expression) => ({ expression })),
});

describe('getHeatmapFieldPlaceholder', () => {
	it.each([
		[DataSource.TRACES, 'Numeric field to bucket, e.g. duration_nano'],
		[DataSource.LOGS, 'Numeric field to bucket'],
	])('%s → %s', (dataSource, expected) => {
		expect(getHeatmapFieldPlaceholder(dataSource)).toBe(expected);
	});
});

describe('withHeatmapField', () => {
	it.each([DataSource.TRACES, DataSource.LOGS])(
		'clears a %s count() for the user to pick a field',
		(dataSource) => {
			expect(
				withHeatmapField(queryOf(dataSource, ['count()'])).aggregations,
			).toStrictEqual([{ expression: '' }]);
		},
	);

	it.each(['attribute.batch_size', 'attribute.queue-depth'])(
		'keeps the bare field %s',
		(field) => {
			const query = queryOf(DataSource.LOGS, [field]);
			expect(withHeatmapField(query)).toBe(query);
		},
	);

	it('keeps the field of a one-argument call', () => {
		expect(
			withHeatmapField(queryOf(DataSource.LOGS, ['avg(attribute.latency)']))
				.aggregations,
		).toStrictEqual([{ expression: 'attribute.latency' }]);
	});

	it.each(['count_distinct(service.name)', 'countIf(status>=500)'])(
		'clears the field rather than bucket the argument of %s',
		(expression) => {
			expect(
				withHeatmapField(queryOf(DataSource.TRACES, [expression])).aggregations,
			).toStrictEqual([{ expression: '' }]);
		},
	);

	it('keeps a field with an alias', () => {
		const query = queryOf(DataSource.LOGS, ['body.latency as lat']);
		expect(withHeatmapField(query)).toBe(query);
	});

	it('renames an order on the aggregation alias to the field', () => {
		const query = {
			...queryOf(DataSource.TRACES, []),
			aggregations: [{ expression: 'p99(duration_nano)', alias: 'lat' }],
			orderBy: [{ columnName: 'lat', order: 'desc' }],
		};
		expect(withHeatmapField(query).orderBy).toStrictEqual([
			{ columnName: 'duration_nano', order: 'desc' },
		]);
	});

	it('renames an order on the aggregation to the field', () => {
		const query = {
			...queryOf(DataSource.TRACES, ['p99(duration_nano)']),
			orderBy: [{ columnName: 'p99(duration_nano)', order: 'desc' }],
		};
		expect(withHeatmapField(query).orderBy).toStrictEqual([
			{ columnName: 'duration_nano', order: 'desc' },
		]);
	});

	it('leaves a metrics query alone', () => {
		const query = initialQueriesMap.metrics.builder.queryData[0];
		expect(withHeatmapField(query)).toBe(query);
	});
});

describe('withoutHeatmapField', () => {
	it.each([
		['duration_nano', 'p90(duration_nano)'],
		['', 'count()'],
	])('turns the field %j into %s', (field, expression) => {
		expect(
			withoutHeatmapField(queryOf(DataSource.TRACES, [field])).aggregations,
		).toStrictEqual([{ expression }]);
	});

	it('drops the alias out of the field', () => {
		expect(
			withoutHeatmapField(queryOf(DataSource.LOGS, ['body.latency as lat']))
				.aggregations,
		).toStrictEqual([{ expression: 'p90(body.latency)' }]);
	});

	it('renames an order on the field to the new aggregation', () => {
		const query = {
			...queryOf(DataSource.TRACES, ['duration_nano']),
			orderBy: [{ columnName: 'duration_nano', order: 'asc' }],
		};
		expect(withoutHeatmapField(query).orderBy).toStrictEqual([
			{ columnName: 'p90(duration_nano)', order: 'asc' },
		]);
	});

	it.each([['count()'], ['count()', 'avg(x)']])(
		'leaves function aggregations %j alone',
		(...expressions) => {
			const query = queryOf(DataSource.LOGS, expressions);
			expect(withoutHeatmapField(query)).toBe(query);
		},
	);
});
