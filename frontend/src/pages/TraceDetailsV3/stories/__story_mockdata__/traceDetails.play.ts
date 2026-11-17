/**
 * AI-owned. Generated and maintained by the `signoz-page-story` skill.
 * Do not hand-edit: regenerate instead.
 */

import { fireEvent, screen, userEvent, within } from 'storybook/test';

import { TraceDetailsTab } from '../../constants';
import {
	listsThread,
	showsTrace,
	startDownload,
	type TraceDetailsArgs,
	untilLoaded,
} from '../TraceDetailsV3.stories.mocks';

interface PlayContext {
	args: Pick<TraceDetailsArgs, 'missingSpans' | 'tab' | 'dataState'>;
}

interface PageContext {
	canvasElement: HTMLElement;
	args: TraceDetailsArgs;
}

/** The header reads the flag off the summary, and only the overview tab shows it. */
export const showsMissingSpansBanner = async ({
	args,
}: PlayContext): Promise<void> => {
	if (
		!args.missingSpans ||
		args.tab !== TraceDetailsTab.Overview ||
		args.dataState !== 'loaded'
	) {
		return;
	}

	await screen.findByText(
		/this trace has missing spans/i,
		undefined,
		untilLoaded,
	);
};

export const showsAiThread = async ({
	canvasElement,
	args,
}: PageContext): Promise<void> => {
	if (!listsThread(args)) {
		return;
	}

	await within(canvasElement).findAllByText(
		/Let me check the upgrade notes/,
		undefined,
		untilLoaded,
	);
};

export const openAnalytics = async ({
	canvasElement,
	args,
}: PageContext): Promise<void> => {
	if (!showsTrace(args)) {
		return;
	}

	await userEvent.click(
		await within(canvasElement).findByRole(
			'button',
			{ name: 'Analytics' },
			untilLoaded,
		),
	);
	await screen.findByTestId('trace-analytics-panel', undefined, untilLoaded);
};

export const showsDownloadPanel = async ({
	canvasElement,
	args,
}: PageContext): Promise<void> => {
	if (!showsTrace(args)) {
		return;
	}

	await startDownload(canvasElement);

	// Only an export that never answers keeps the panel up.
	if (args.exportState === 'loading') {
		await screen.findByTestId('trace-download-panel', undefined, untilLoaded);
	}
};

const openFunnelPicker = async (canvasElement: HTMLElement): Promise<void> => {
	const [button] = await within(canvasElement).findAllByTestId(
		'span-add-to-funnel-btn',
		undefined,
		untilLoaded,
	);

	await fireEvent.click(button);
	await screen.findByText('Add span to funnel', undefined, untilLoaded);
};

export const openAddSpanToFunnel = async ({
	canvasElement,
	args,
}: PageContext): Promise<void> => {
	if (!showsTrace(args)) {
		return;
	}

	await openFunnelPicker(canvasElement);
};

export const openCreateFunnelStep = async ({
	canvasElement,
	args,
}: PageContext): Promise<void> => {
	if (!showsTrace(args)) {
		return;
	}

	await openFunnelPicker(canvasElement);
	await userEvent.click(
		await screen.findByRole('button', { name: /create new funnel/i }),
	);
	await screen.findByTestId('create-funnel-name-input');
};
