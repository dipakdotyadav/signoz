/**
 * AI-owned. Generated and maintained by the `signoz-page-story` skill.
 * Do not hand-edit: regenerate instead.
 */

import { screen, userEvent, within } from 'storybook/test';

import {
	openConfigSelect,
	type PanelEditorArgs,
	savePanel,
} from '../PanelEditorPage.stories.mocks';

interface PlayContext {
	canvasElement: HTMLElement;
	args: PanelEditorArgs;
}

const PAGE_LOAD = { timeout: 10000 };

export const expandConfigSections = async ({
	canvasElement,
}: PlayContext): Promise<void> => {
	const canvas = within(canvasElement);

	const headers = await canvas.findAllByTestId(
		/^config-section-/,
		{},
		PAGE_LOAD,
	);
	for (const header of headers) {
		if (header.getAttribute('aria-expanded') === 'false') {
			await userEvent.click(header);
		}
	}
	(document.activeElement as HTMLElement | null)?.blur();
	for (let el = headers[0].parentElement; el; el = el.parentElement) {
		el.scrollTop = 0;
	}
};

export const openAddColumn = async ({ args }: PlayContext): Promise<void> => {
	if (args.panel !== 'recent-logs') {
		return;
	}

	await userEvent.click(
		await screen.findByTestId('list-columns-add', undefined, PAGE_LOAD),
	);
	await screen.findAllByTestId('list-columns-suggestion');
};

export const openTypeSwitcher = async ({
	canvasElement,
}: PlayContext): Promise<void> => {
	await userEvent.click(
		await within(canvasElement).findByTestId(
			'panel-editor-v2-type-switcher',
			{},
			PAGE_LOAD,
		),
	);
	await screen.findByTestId('panel-type-switcher-drawer');
};

export const changeConfig = async (context: PlayContext): Promise<void> => {
	const canvas = within(context.canvasElement);

	await expandConfigSections(context);
	await userEvent.click(
		await canvas.findByTestId('panel-editor-v2-log-scale-log'),
	);
	await userEvent.click(
		await canvas.findByTestId('panel-editor-v2-line-style-dashed'),
	);
};

export const saveAndToast = async ({ args }: PlayContext): Promise<void> => {
	await savePanel();
	if (args.panelSave === 'success') {
		await screen.findByText('Panel saved');
	}
};

/** Switches a new heatmap to traces and opens its field picker. */
export const pickHeatmapTracesField = async ({
	args,
}: PlayContext): Promise<void> => {
	if (args.panel !== 'new' || args.newPanelKind !== 'heatmap') {
		return;
	}

	await openConfigSelect('query-data-source-selector-0');
	await userEvent.click(await screen.findByTitle('Traces'));

	await openConfigSelect('heatmap-field-select');
	if (args.dataState === 'loaded') {
		await screen.findByTitle('attribute.retry.count', undefined, PAGE_LOAD);
	}
};
