import type { PANEL_TYPES } from 'constants/queryBuilder';
import {
	IBuilderQuery,
	OrderByPayload,
} from 'types/api/queryBuilder/queryBuilderData';

export type OrderByFilterProps = {
	query: IBuilderQuery;
	onChange: (values: OrderByPayload[]) => void;
	isRawQuery?: boolean;
	entityVersion?: string;
	isNewQueryV2?: boolean;
	placeholder?: string;
	/** A heatmap's aggregation is a bare field, which Order by then offers. */
	panelType?: PANEL_TYPES | null;
};

export type OrderByFilterValue = {
	disabled?: boolean;
	key: string;
	label: string;
	title?: string;
	value: string;
};
