/**
 * AI-owned. Generated and maintained by the `signoz-page-story` skill.
 * Do not hand-edit: regenerate instead.
 */

import { screen, userEvent, within } from 'storybook/test';

import {
	expandSection,
	hasClickHouseTab,
	hasDisconnectValues,
	type PanelEditorArgs,
	pickSegment,
} from '../PanelEditorPage.stories.mocks';

interface PlayContext {
	canvasElement: HTMLElement;
	args: PanelEditorArgs;
}

export const openClickHouseQuery = async ({
	canvasElement,
	args,
}: PlayContext): Promise<void> => {
	if (!hasClickHouseTab(args)) {
		return;
	}

	await userEvent.click(
		await within(canvasElement).findByText('ClickHouse Query', undefined, {
			timeout: 15000,
		}),
	);
	await screen.findByText(/ClickHouse query agent skill/i, undefined, {
		timeout: 15000,
	});
};

export const enterInvalidDisconnectThreshold = async ({
	args,
}: PlayContext): Promise<void> => {
	if (!hasDisconnectValues(args)) {
		return;
	}

	await expandSection('Chart appearance');
	await screen.findByTestId('panel-editor-v2-line-style');
	await pickSegment('panel-editor-v2-span-gaps', 'threshold');
	const input = await screen.findByTestId('panel-editor-v2-span-gaps-value');

	await userEvent.clear(input);
	await userEvent.type(input, '5x');
	await screen.findByText(/enter a valid duration/i);
};
