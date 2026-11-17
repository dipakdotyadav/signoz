/**
 * AI-owned. Generated and maintained by the `signoz-page-story` skill.
 * Do not hand-edit: regenerate instead.
 */

import { screen, userEvent, within } from 'storybook/test';

import type { PageStoryArgs } from '@/storybook/runtime/resolveStory';

import type { pipelinesMocks } from '../Pipelines.stories.mocks';

type PipelinesArgs = PageStoryArgs<typeof pipelinesMocks>;

interface PlayContext {
	canvasElement: HTMLElement;
	args: Pick<PipelinesArgs, 'pipelines' | 'dataState'>;
}

/** The page fetches before it renders a row, which outlasts the 1s default. */
export const untilLoaded = { timeout: 15_000 };

/** Row actions repeat per pipeline, so a story that opens one takes the first. */
export const clickFirst = async (
	canvasElement: HTMLElement,
	testId: string,
): Promise<void> => {
	const [action] = await within(canvasElement).findAllByTestId(
		testId,
		undefined,
		untilLoaded,
	);

	await userEvent.click(action);
};

/**
 * Editing is a mode the whole table is in: the row actions, the drag handles and
 * the save bar only exist inside it.
 */
export const enterEditMode = async (
	canvasElement: HTMLElement,
): Promise<void> => {
	await userEvent.click(
		await within(canvasElement).findByText(
			/enter edit mode/i,
			undefined,
			untilLoaded,
		),
	);
};

export const addSeverityParserProcessor = async ({
	canvasElement,
	args,
}: PlayContext): Promise<void> => {
	if (args.dataState !== 'loaded' || args.pipelines === 0) {
		return;
	}

	await enterEditMode(canvasElement);
	await clickFirst(canvasElement, 'pipeline-row-expand');
	await userEvent.click(await screen.findByText(/add processor/i));
	await screen.findByText(/create new processor/i);
	await userEvent.click(await screen.findByText('Grok'));
	await userEvent.click(await screen.findByTitle('Severity Parser'));
	await screen.findByText('Parse Severity Value From');
};
